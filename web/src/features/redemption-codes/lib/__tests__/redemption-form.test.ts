import { describe, expect, test } from 'vitest'

import {
  transformFormDataToPayload,
  transformRedemptionToFormDefaults,
  type RedemptionFormValues,
} from '../redemption-form'
import type { Redemption } from '../../types'

function formValues(
  overrides: Partial<RedemptionFormValues> = {}
): RedemptionFormValues {
  return {
    name: 'gift',
    reward_type: 'quota',
    quota_dollars: 10,
    expired_time: undefined,
    count: 2,
    max_uses: 1,
    max_uses_per_user: 1,
    ...overrides,
  }
}

describe('redemption form payload', () => {
  test('quota codes send a positive quota and no plan', () => {
    expect(transformFormDataToPayload(formValues())).toEqual({
      name: 'gift',
      quota: 5000000,
      expired_time: 0,
      count: 2,
      plan_id: 0,
      max_uses: 1,
      max_uses_per_user: 1,
    })
  })

  test('subscription codes send quota 0 and the selected plan', () => {
    expect(
      transformFormDataToPayload(
        formValues({
          reward_type: 'subscription',
          plan_id: 42,
          max_uses: 10,
          max_uses_per_user: 0,
        })
      )
    ).toEqual({
      name: 'gift',
      quota: 0,
      expired_time: 0,
      count: 2,
      plan_id: 42,
      max_uses: 10,
      max_uses_per_user: 0,
    })
  })

  test('loads a subscription code back into subscription form values', () => {
    const redemption: Redemption = {
      id: 8,
      user_id: 1,
      name: 'sub',
      key: 'abc',
      status: 1,
      quota: 0,
      created_time: 1,
      redeemed_time: 0,
      expired_time: 0,
      used_user_id: 0,
      plan_id: 9,
      plan_title: 'Pro',
      max_uses: 0,
      max_uses_per_user: 2,
      used_count: 3,
    }

    const values = transformRedemptionToFormDefaults(redemption)
    expect(values.reward_type).toBe('subscription')
    expect(values.plan_id).toBe(9)
    expect(values.quota_dollars).toBe(0)
    expect(values.max_uses).toBe(0)
    expect(values.max_uses_per_user).toBe(2)
  })
})
