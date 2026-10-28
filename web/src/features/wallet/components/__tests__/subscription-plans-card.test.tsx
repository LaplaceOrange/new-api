import { render, screen } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import {
  getPublicPlans,
  getSelfSubscriptionFull,
} from '@/features/subscriptions/api'

import { SubscriptionPlansCard } from '../subscription-plans-card'

vi.mock('@/features/subscriptions/api', () => ({
  getPublicPlans: vi.fn(),
  getSelfSubscriptionFull: vi.fn(),
  updateBillingPreference: vi.fn(),
  updateSubscriptionPriority: vi.fn(),
}))

vi.mock(
  '@/features/subscriptions/components/dialogs/subscription-purchase-dialog',
  () => ({
    SubscriptionPurchaseDialog: () => null,
  })
)

const subscription = {
  id: 31,
  user_id: 7,
  plan_id: 8,
  status: 'active',
  start_time: 1,
  end_time: Math.floor(Date.now() / 1000) + 86400,
  amount_total: 0,
  amount_used: 50_000,
}

function setSelfSubscription(concurrencyLimit: number) {
  vi.mocked(getSelfSubscriptionFull).mockResolvedValue({
    success: true,
    data: {
      billing_preference: 'subscription_first',
      subscriptions: [{ subscription }],
      all_subscriptions: [{ subscription }],
      token_usages: {
        '31': {
          enabled: true,
          quota_enabled: true,
          mode: 'shared',
          total: 0,
          used: 100,
        },
      },
      concurrency_usages: {
        '31': {
          current: 2,
          limit: concurrencyLimit,
          known: true,
          limit_known: true,
        },
      },
    },
  })
}

beforeEach(() => {
  vi.mocked(getPublicPlans).mockResolvedValue({ success: true, data: [] })
})

it('shows used and unlimited remaining quota, tokens, and concurrency', async () => {
  setSelfSubscription(0)
  render(<SubscriptionPlansCard topupInfo={null} />)

  expect(
    await screen.findByText(
      (_, element) =>
        element?.childElementCount === 0 &&
        (element.textContent?.includes('Total Quota:') ?? false) &&
        (element.textContent?.includes('/Unlimited · Remaining Unlimited') ??
          false)
    )
  ).toBeInTheDocument()
  expect(
    screen.getByText(/100\/Unlimited · Remaining Unlimited/)
  ).toBeInTheDocument()
  expect(
    screen.getByText(
      (_, element) =>
        element?.childElementCount === 0 &&
        (element.textContent?.includes('Current Concurrency: 2/Unlimited') ??
          false)
    )
  ).toBeInTheDocument()
})

it('shows the configured concurrency cap beside the current routed requests', async () => {
  setSelfSubscription(3)
  render(<SubscriptionPlansCard topupInfo={null} />)

  expect(
    await screen.findByText(
      (_, element) =>
        element?.childElementCount === 0 &&
        (element.textContent?.includes('Current Concurrency: 2/3') ?? false)
    )
  ).toBeInTheDocument()
})
