import { render, screen } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { expect, it } from 'vitest'

import { RechargeFormCard } from '../recharge-form-card'

const props: ComponentProps<typeof RechargeFormCard> = {
  topupInfo: {
    enable_online_topup: true,
    enable_stripe_topup: false,
    pay_methods: [],
    min_topup: 1,
    stripe_min_topup: 1,
    amount_options: [],
    discount: {},
  },
  presetAmounts: [],
  selectedPreset: null,
  onSelectPreset: () => {},
  topupAmount: 100,
  onTopupAmountChange: () => {},
  paymentAmount: 108,
  calculating: false,
  onPaymentMethodSelect: () => {},
  paymentLoading: null,
  redemptionCode: '',
  onRedemptionCodeChange: () => {},
  onRedeem: () => {},
  redeeming: false,
}
it('shows the included fee beside the payable amount and hides it while recalculating', () => {
  const view = render(<RechargeFormCard {...props} paymentFee={8} />)
  expect(screen.getByText('108')).toBeInTheDocument()
  expect(screen.getByText('Includes 8 payment fee')).toBeInTheDocument()
  view.rerender(<RechargeFormCard {...props} paymentFee={8} calculating />)
  expect(screen.queryByText('Includes 8 payment fee')).not.toBeInTheDocument()
})
it('does not label a fee-free quote as containing fees', () => {
  render(<RechargeFormCard {...props} paymentFee={0} />)
  expect(screen.queryByText(/Includes/)).not.toBeInTheDocument()
})
