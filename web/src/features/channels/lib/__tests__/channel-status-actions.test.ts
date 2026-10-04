import type { QueryClient } from '@tanstack/react-query'
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
import { afterEach, assert, expect, test, vi } from 'vitest'

const { api } = await import('@/lib/api')
const { handleBatchEnable, handleTestChannel } =
  await import('../channel-actions')
const { toast } = await import('sonner')

type ApiPost = (url: string, data?: unknown) => Promise<{ data: unknown }>
const apiClient = api as unknown as { post: ApiPost }
const originalPost = apiClient.post

afterEach(() => {
  apiClient.post = originalPost
  vi.restoreAllMocks()
})

test('successful connectivity reports price disabling independently to the caller', async () => {
  const priceMonitor = {
    checked: true,
    rate_multiplier: 2,
    limit: 1,
    exceeded: true,
    disabled: true,
    enabled: false,
  }
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, time: 0.1, price_monitor: priceMonitor },
  })
  const warning = vi.spyOn(toast, 'warning')
  const callback = vi.fn()
  await handleTestChannel(42, { channelName: 'Channel A' }, callback)
  expect(callback).toHaveBeenCalledWith(
    true,
    100,
    undefined,
    undefined,
    priceMonitor
  )
  expect(warning).toHaveBeenCalledWith(
    'Channel disabled by price protection: 2 > 1'
  )
})

test('price recovery remains available on silent tests without showing notifications', async () => {
  const priceMonitor = {
    checked: true,
    rate_multiplier: 0.5,
    limit: 1,
    exceeded: false,
    disabled: false,
    enabled: true,
  }
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, time: 0.1, price_monitor: priceMonitor },
  })
  const success = vi.spyOn(toast, 'success')
  const callback = vi.fn()
  await handleTestChannel(42, { silent: true }, callback)
  expect(callback).toHaveBeenCalledWith(
    true,
    100,
    undefined,
    undefined,
    priceMonitor
  )
  expect(success).not.toHaveBeenCalled()
})

test('refreshes channels after a partially successful batch status update', async () => {
  const invalidatedKeys: unknown[] = []
  let callbackCalled = false
  apiClient.post = async (url, data) => {
    assert.equal(url, '/api/channel/status/batch')
    assert.deepEqual(data, { ids: [11, 12], status: 1 })
    return {
      data: {
        success: false,
        message: 'failed to update channel status for ids: [12]',
        data: { changed: 1, failed_ids: [12] },
      },
    }
  }
  const queryClient = {
    invalidateQueries: async (options: { queryKey: unknown }) => {
      invalidatedKeys.push(options.queryKey)
    },
  } as unknown as QueryClient

  await handleBatchEnable([11, 12], queryClient, () => {
    callbackCalled = true
  })

  assert.deepEqual(invalidatedKeys, [['channels'], ['models']])
  assert.equal(callbackCalled, true)
})
