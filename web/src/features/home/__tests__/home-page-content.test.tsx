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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { useHomePageContent } from '../hooks/use-home-page-content'

let queryClient: QueryClient

function Wrapper(props: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      {props.children}
    </QueryClientProvider>
  )
}

beforeEach(() => {
  localStorage.clear()
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
})

afterEach(() => {
  queryClient.clear()
  localStorage.clear()
})

it.each(['# Cached home', ''])(
  'displays cached content %j while its refresh is pending',
  async (cached) => {
    localStorage.setItem('home_page_content', cached)
    let resolve!: (value: { data: unknown }) => void
    vi.spyOn(api, 'get').mockReturnValue(
      new Promise((done) => {
        resolve = done
      })
    )
    const hook = renderHook(useHomePageContent, { wrapper: Wrapper })

    expect(hook.result.current).toMatchObject({
      content: cached,
      isLoaded: true,
    })

    await act(async () => {
      resolve({ data: { success: true, data: '# Updated home' } })
    })
    await waitFor(() =>
      expect(hook.result.current.content).toBe('# Updated home')
    )
    expect(localStorage.getItem('home_page_content')).toBe('# Updated home')
  }
)

it('preserves cached content when the server reports a failed refresh', async () => {
  localStorage.setItem('home_page_content', '# Cached home')
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: false, message: 'Home content unavailable' },
  })
  const hook = renderHook(useHomePageContent, { wrapper: Wrapper })

  await waitFor(() => expect(hook.result.current.isLoaded).toBe(true))
  expect(hook.result.current.content).toBe('# Cached home')
  expect(localStorage.getItem('home_page_content')).toBe('# Cached home')
})

it('shows fetched content even when browser storage is unavailable', async () => {
  vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
    throw new Error('Storage unavailable')
  })
  vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
    throw new Error('Storage unavailable')
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: 'https://example.com/home' },
  })
  const hook = renderHook(useHomePageContent, { wrapper: Wrapper })

  await waitFor(() =>
    expect(hook.result.current).toEqual({
      content: 'https://example.com/home',
      isLoaded: true,
      isUrl: true,
    })
  )
})
