import { describe, expect, it } from 'vitest'

import {
  formValuesToPlanPayload,
  PLAN_FORM_DEFAULTS,
  planToFormValues,
} from '../plan-form'

describe('subscription plan form benefit mapping', () => {
  it('submits quota-only plans without token benefits', () => {
    const payload = formValuesToPlanPayload({
      ...PLAN_FORM_DEFAULTS,
      content_type: 'quota',
      total_amount: 10,
      token_amount: 500,
    })

    expect(payload.plan.content_type).toBe('quota')
    expect(payload.plan.total_amount).toBeGreaterThan(0)
    expect(payload.plan.token_amount).toBe(0)
  })

  it('submits token-only plans without quota benefits', () => {
    const payload = formValuesToPlanPayload({
      ...PLAN_FORM_DEFAULTS,
      content_type: 'token',
      total_amount: 10,
      token_amount: 500,
    })

    expect(payload.plan.content_type).toBe('token')
    expect(payload.plan.total_amount).toBe(0)
    expect(payload.plan.token_amount).toBe(500)
  })

  it('keeps legacy combined plans editable without changing their stored type', () => {
    const values = planToFormValues({
      id: 1,
      title: 'Legacy plan',
      price_amount: 0,
      currency: 'USD',
      duration_unit: 'month',
      duration_value: 1,
      quota_reset_period: 'never',
      enabled: true,
      sort_order: 0,
      allow_balance_pay: true,
      allow_wallet_overflow: true,
      max_purchase_per_user: 0,
      total_amount: 50_000,
      content_type: '',
      token_amount: 500,
      token_count_mode: 'shared',
      token_groups: '',
      token_models: '',
      max_concurrency: 0,
    })
    const payload = formValuesToPlanPayload(values)

    expect(values.content_type).toBe('legacy')
    expect(payload.plan.content_type).toBe('')
    expect(payload.plan.total_amount).toBe(50_000)
    expect(payload.plan.token_amount).toBe(500)
  })
})
