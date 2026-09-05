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
import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, describe, expect, test } from 'vitest'

import { PaymentConfirmDialog } from '../payment-confirm-dialog'

beforeAll(() => {
  i18next.addResourceBundle('en', 'translation', {
    '(Includes {{fee}} payment fee)': '(Includes {{fee}} payment fee)',
  })
})

describe('PaymentConfirmDialog', () => {
  test('shows the quoted total and payment fee when the fee is non-zero', () => {
    render(
      <PaymentConfirmDialog
        open
        onOpenChange={() => undefined}
        onConfirm={() => undefined}
        topupAmount={100}
        paymentAmount={100}
        quote={{
          subtotal: '100.00',
          fee: '8.00',
          total: '108.00',
          fee_rate: '8.00',
        }}
        paymentMethod={{ name: 'Stripe', type: 'stripe' }}
        calculating={false}
        processing={false}
      />
    )

    expect(screen.getByText('108')).toBeInTheDocument()
    expect(screen.getByText('(Includes 8 payment fee)')).toBeInTheDocument()
  })

  test('hides the payment fee description when the fee is zero', () => {
    render(
      <PaymentConfirmDialog
        open
        onOpenChange={() => undefined}
        onConfirm={() => undefined}
        topupAmount={100}
        paymentAmount={100}
        quote={{
          subtotal: '100.00',
          fee: '0.00',
          total: '100.00',
          fee_rate: '0.00',
        }}
        paymentMethod={{ name: 'Stripe', type: 'stripe' }}
        calculating={false}
        processing={false}
      />
    )

    expect(screen.queryByText(/payment fee/)).not.toBeInTheDocument()
  })
})
