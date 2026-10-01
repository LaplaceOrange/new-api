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
import { api } from '@/lib/http-client'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'

import type {
  ReferralAdminData,
  ReferralBucket,
  ReferralCampaign,
  ReferralPolicy,
  ReferralRewards,
} from './types'

interface ReferralResponse<T> {
  success: boolean
  message?: string
  data?: T
}

function referralData<T>(response: ReferralResponse<T>): T {
  requireServerSuccess(response)
  if (response.data == null) {
    throw createServerError(response, 'Unable to load referral data')
  }
  return response.data
}

export async function getReferralRewards(page = 1): Promise<ReferralRewards> {
  return referralData(
    (
      await api.get<ReferralResponse<ReferralRewards>>(
        '/api/referrals/rewards',
        {
          params: { p: page, page_size: 20 },
        }
      )
    ).data
  )
}

export async function getReferralAdmin(): Promise<ReferralAdminData> {
  return referralData(
    (await api.get<ReferralResponse<ReferralAdminData>>('/api/referrals/admin'))
      .data
  )
}

export async function saveReferralCampaign(
  campaign: ReferralCampaign
): Promise<ReferralCampaign> {
  const path = '/api/referrals/admin/campaigns'
  const response =
    campaign.id > 0
      ? await api.put<ReferralResponse<ReferralCampaign>>(
          `${path}/${campaign.id}`,
          campaign
        )
      : await api.post<ReferralResponse<ReferralCampaign>>(path, campaign)
  return referralData(response.data)
}

export async function saveReferralPolicy(
  policy: ReferralPolicy
): Promise<ReferralPolicy> {
  return referralData(
    (
      await api.put<ReferralResponse<ReferralPolicy>>(
        '/api/referrals/admin/policy',
        policy
      )
    ).data
  )
}

export async function getReferralBuckets(
  startAt: number,
  endAt: number
): Promise<ReferralBucket[]> {
  return referralData(
    (
      await api.get<ReferralResponse<ReferralBucket[]>>(
        '/api/referrals/admin/withdrawals',
        {
          params: { start_at: startAt, end_at: endAt },
        }
      )
    ).data
  )
}
