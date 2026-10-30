/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient } from '@tanstack/react-query'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { STATUS_QUERY_KEY, statusQueryOptions } from '@/lib/status-query'

import { getSetupComplete } from '../api'

let queryClient: QueryClient

beforeEach(() => {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
})

afterEach(() => {
  queryClient.clear()
  localStorage.clear()
})

it.each([true, false])(
  'shares the pending status request and preserves setup=%s',
  async (setup) => {
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { setup } },
    })
    const [status, complete] = await Promise.all([
      queryClient.fetchQuery(statusQueryOptions),
      getSetupComplete(queryClient),
    ])
    expect(status?.setup).toBe(setup)
    expect(complete).toBe(setup)
    expect(get).toHaveBeenCalledTimes(1)
    expect(get).toHaveBeenCalledWith('/api/status')
  }
)

it('refreshes a stale setup flag before deciding whether setup is complete', async () => {
  queryClient.setQueryData(
    STATUS_QUERY_KEY,
    { setup: true },
    { updatedAt: Date.now() - 600_000 }
  )
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { setup: false } },
  })
  expect(await getSetupComplete(queryClient)).toBe(false)
})

it.each(['missing setup field', 'status request failure'])(
  'falls back to the setup endpoint on %s',
  async (scenario) => {
    const get = vi.spyOn(api, 'get').mockImplementation(async (url) => {
      if (url === '/api/status') {
        if (scenario === 'status request failure') throw new Error('Offline')
        return { data: { success: true, data: {} } }
      }
      return { data: { success: true, data: { status: false } } }
    })
    expect(await getSetupComplete(queryClient)).toBe(false)
    expect(get.mock.calls.map(([url]) => url)).toEqual([
      '/api/status',
      '/api/setup',
    ])
  }
)
