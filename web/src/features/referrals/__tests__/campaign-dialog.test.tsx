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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, test, vi } from 'vitest'

import { CampaignDialog } from '../campaign-dialog'
import type { ReferralCampaign } from '../types'

const campaign: ReferralCampaign = {
  id: 0,
  name: 'Autumn',
  enabled: false,
  priority: 1,
  start_at: 100,
  end_at: 200,
  direction: 'both',
  spend_basis: 'wallet',
  required_friends: 5,
  paid_threshold_quota: 500_000,
  base_bps: 200,
  inviter_bps: 500,
  invitee_pool_bps: 400,
}

test('keeps new campaigns disabled and submits editable rates as integer basis points', async () => {
  const save = vi.fn(async (_value: ReferralCampaign) => undefined)
  render(
    <CampaignDialog
      campaign={campaign}
      pending={false}
      onClose={() => undefined}
      onSave={save}
    />
  )
  expect(screen.getByRole('switch', { name: 'Enabled' })).toHaveAttribute(
    'aria-disabled',
    'true'
  )
  expect(screen.getByRole('switch', { name: 'Enabled' })).not.toBeChecked()
  fireEvent.change(screen.getByLabelText('Base rebate (%)'), {
    target: { value: '2.32' },
  })
  fireEvent.change(screen.getByLabelText('Required qualified friends'), {
    target: { value: '3' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(save).toHaveBeenCalled())
  expect(save.mock.calls[0][0]).toMatchObject({
    enabled: false,
    base_bps: 232,
    required_friends: 3,
  })
})

test('shows validation and does not submit a campaign without a name', async () => {
  const save = vi.fn(async (_value: ReferralCampaign) => undefined)
  render(
    <CampaignDialog
      campaign={{ ...campaign, name: '' }}
      pending={false}
      onClose={() => undefined}
      onSave={save}
    />
  )
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(screen.getByLabelText('Name')).toHaveAttribute(
      'aria-invalid',
      'true'
    )
  )
  expect(save).not.toHaveBeenCalled()
})

test('clearing the labelled start time preserves the end time and prevents submission', async () => {
  const save = vi.fn(async (_value: ReferralCampaign) => undefined)
  render(
    <CampaignDialog
      campaign={campaign}
      pending={false}
      onClose={() => undefined}
      onSave={save}
    />
  )
  expect(screen.getByRole('button', { name: /^End time$/ })).toHaveTextContent(
    '1970-01-01'
  )
  fireEvent.click(screen.getByRole('button', { name: 'Start time Clear' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(
      screen.getAllByText('End time must be after start time').length
    ).toBeGreaterThan(0)
  )
  expect(screen.getByRole('button', { name: /^End time$/ })).toHaveTextContent(
    '1970-01-01'
  )
  expect(save).not.toHaveBeenCalled()
})
