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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

import { api } from '@/lib/http-client'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { ContributionRewards } from '../contribution-rewards'

type ApiMethod = (
  url: string,
  data?: unknown,
  config?: unknown
) => Promise<{ data: unknown }>

type MockableApi = {
  get: ApiMethod
  post: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post

function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

function renderRewards(queryClient: QueryClient): void {
  render(
    <QueryClientProvider client={queryClient}>
      <ContributionRewards rewardBps={500} />
    </QueryClientProvider>
  )
}

function configureCurrency(): void {
  useSystemConfigStore.getState().setConfig({
    currency: {
      displayInCurrency: true,
      quotaDisplayType: 'USD',
      quotaPerUnit: 500000,
      usdExchangeRate: 1,
      customCurrencySymbol: '¤',
      customCurrencyExchangeRate: 1,
    },
  })
}

function mockRewardQueries(): void {
  apiClient.get = async (url) => {
    if (url.endsWith('/rewards')) {
      return {
        data: {
          success: true,
          data: {
            account: {
              user_id: 42,
              balance: 500000,
              lifetime_earned: 500000,
              lifetime_transferred: 0,
            },
            items: [],
            total: 0,
          },
        },
      }
    }
    return { data: { success: true, data: { items: [], total: 0 } } }
  }
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
  useSystemConfigStore.getState().setConfig({
    currency: {
      displayInCurrency: true,
      quotaDisplayType: 'USD',
      quotaPerUnit: 500000,
      usdExchangeRate: 1,
      customCurrencySymbol: '¤',
      customCurrencyExchangeRate: 1,
    },
  })
  localStorage.clear()
})

describe('channel contribution reward transfer', () => {
  test('uses the configured display amount for the transfer input', async () => {
    configureCurrency()
    mockRewardQueries()
    const queryClient = createQueryClient()

    renderRewards(queryClient)

    const input = await screen.findByRole('spinbutton', {
      name: 'Transfer amount',
    })
    await waitFor(() => expect(input).toHaveAttribute('max', '1'))
    expect(Number(input.getAttribute('step'))).toBeCloseTo(0.000002)
    expect(input).toHaveAttribute('placeholder', 'Enter amount in USD')
  })

  test('converts the entered display amount to quota before submitting', async () => {
    configureCurrency()
    mockRewardQueries()
    const transfers: unknown[] = []
    apiClient.post = async (_url, data) => {
      transfers.push(data)
      return {
        data: {
          success: true,
          data: { id: 1, amount: -500000, balance_after: 0 },
        },
      }
    }
    const queryClient = createQueryClient()

    renderRewards(queryClient)

    const input = await screen.findByRole('spinbutton', {
      name: 'Transfer amount',
    })
    await waitFor(() => expect(input).toHaveAttribute('max', '1'))
    fireEvent.change(input, { target: { value: '1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Transfer to wallet' }))

    await waitFor(() => expect(transfers).toEqual([{ amount: 500000 }]))
  })

  test('fills transfer all with the configured display amount', async () => {
    configureCurrency()
    mockRewardQueries()
    const queryClient = createQueryClient()

    renderRewards(queryClient)

    const input = await screen.findByRole('spinbutton', {
      name: 'Transfer amount',
    })
    await waitFor(() => expect(input).toHaveAttribute('max', '1'))
    fireEvent.click(screen.getByRole('button', { name: 'Transfer all' }))

    expect(input).toHaveValue(1)
  })
})
