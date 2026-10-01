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
import { describe, expect, test } from 'vitest'

import { referralCampaignSchema, referralPolicySchema } from '../form-schema'
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

describe('referral campaign validation', () => {
  test.each([
    ['empty name', { name: '' }],
    ['equal time boundaries', { end_at: 100 }],
    ['reversed time boundaries', { end_at: 99 }],
    ['fractional friend count', { required_friends: 1.5 }],
    ['fractional basis points', { base_bps: 200.5 }],
    ['rate above 100 percent', { inviter_bps: 10_001 }],
    ['negative credit threshold', { paid_threshold_quota: -1 }],
  ])('rejects %s', (_name, fields) => {
    expect(
      referralCampaignSchema.safeParse({ ...campaign, ...fields }).success
    ).toBe(false)
  })

  test('accepts configurable directions, bases, zero threshold and bounded rates', () => {
    const edited = {
      ...campaign,
      direction: 'inviter_to_invitee',
      spend_basis: 'paid_wallet',
      paid_threshold_quota: 0,
      base_bps: 0,
      inviter_bps: 10_000,
    }
    expect(referralCampaignSchema.parse(edited)).toEqual(edited)
  })
})

describe('referral withdrawal policy validation', () => {
  test.each([0, -1, 0.5, Infinity])(
    'rejects non-positive or non-integer interval %s',
    (minutes) => {
      expect(
        referralPolicySchema.safeParse({
          min_transfer_quota: 123,
          bucket_minutes: minutes,
        }).success
      ).toBe(false)
    }
  )
  test('keeps withdrawal quota independent of the positive integer minute bucket', () => {
    expect(
      referralPolicySchema.parse({ min_transfer_quota: 123, bucket_minutes: 7 })
    ).toEqual({ min_transfer_quota: 123, bucket_minutes: 7 })
  })
})
