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
import { fireEvent, render, screen } from '@testing-library/react'
import { expect, test, vi } from 'vitest'

import { ReferralRewardHistory } from '../reward-history'
import type { ReferralRewards } from '../types'

const rewards: ReferralRewards = {
  summary: {
    balance: 20,
    legacy_balance: 0,
    lifetime_earned: 20,
    min_transfer_quota: 1,
    campaign: null,
    invited_count: 0,
    qualified_count: 0,
    unlocked: false,
  },
  entries: [
    {
      id: 1,
      campaign_id: 7,
      source_kind: 'wallet',
      source_id: 'request-1',
      entry_kind: 'invitee',
      amount: 20,
      source_quota: 1_000,
      rate_bps: 400,
      pool_recipients: 2,
      pool_index: 0,
      created_at: 1_790_000_000,
      campaign_snapshot: '{"name":"Frozen autumn"}',
    },
  ],
  transfers: [
    {
      id: 1,
      amount: 10,
      legacy_part: 0,
      created_at: 1_790_000_000,
      source_snapshot: '[{"campaign_id":7,"source_kind":"wallet","amount":10}]',
    },
  ],
  entry_total: 25,
  transfer_total: 1,
}

test('shows frozen campaign parameters and uses shared pagination for each history view', () => {
  const change = vi.fn()
  render(
    <ReferralRewardHistory
      rewards={rewards}
      page={1}
      pending={false}
      onPageChange={change}
    />
  )
  expect(screen.getByText('Frozen autumn')).toBeVisible()
  expect(screen.getByText('4% / 2')).toBeVisible()
  fireEvent.click(screen.getByRole('button', { name: 'Go to next page' }))
  expect(change).toHaveBeenLastCalledWith(2)
  fireEvent.click(screen.getByRole('tab', { name: 'Withdrawal history' }))
  expect(change).toHaveBeenLastCalledWith(1)
  expect(screen.getByRole('button', { name: 'Go to next page' })).toBeDisabled()
})

test('blocks pagination while the next history page is loading', () => {
  const change = vi.fn()
  render(
    <ReferralRewardHistory
      rewards={rewards}
      page={1}
      pending
      onPageChange={change}
    />
  )
  expect(screen.getByRole('button', { name: 'Go to next page' })).toBeDisabled()
})
