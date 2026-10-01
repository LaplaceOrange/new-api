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
import { afterEach, expect, test, vi } from 'vitest'

import { useSystemConfigStore } from '@/stores/system-config-store'

import { TransferDialog } from '../transfer-dialog'

const oldConfig = useSystemConfigStore.getState().config
afterEach(() => useSystemConfigStore.getState().setConfig(oldConfig))

test('accepts a partial transfer above the server minimum without an amount step restriction', async () => {
  useSystemConfigStore.getState().setConfig({
    currency: {
      ...oldConfig.currency,
      quotaPerUnit: 500_000,
      quotaDisplayType: 'USD',
      usdExchangeRate: 1,
    },
  })
  const confirm = vi.fn(async (_quota: number) => false)
  const close = vi.fn()
  render(
    <TransferDialog
      open
      onOpenChange={close}
      onConfirm={confirm}
      availableQuota={500_000}
      minimumQuota={125_000}
      transferring={false}
    />
  )
  const input = screen.getByLabelText('Transfer Amount')
  expect(input).toHaveAttribute('step', 'any')
  fireEvent.change(input, { target: { value: '0.37' } })
  fireEvent.click(screen.getByRole('button', { name: /^Transfer$/ }))
  await waitFor(() => expect(confirm).toHaveBeenCalledWith(185_000))
  expect(close).not.toHaveBeenCalled()
  expect(screen.getByRole('dialog')).toBeVisible()
})

test('disables transfers when net reward balance is negative', () => {
  render(
    <TransferDialog
      open
      onOpenChange={() => undefined}
      onConfirm={async () => true}
      availableQuota={-50}
      minimumQuota={1}
      transferring={false}
    />
  )
  expect(screen.getByRole('button', { name: /^Transfer$/ })).toBeDisabled()
})
