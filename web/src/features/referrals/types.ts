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
export interface ReferralCampaign {
  id: number
  name: string
  enabled: boolean
  priority: number
  start_at: number
  end_at: number
  direction: 'invitee_to_inviter' | 'inviter_to_invitee' | 'both'
  spend_basis: 'wallet' | 'all' | 'paid_wallet'
  required_friends: number
  paid_threshold_quota: number
  base_bps: number
  inviter_bps: number
  invitee_pool_bps: number
}

export interface ReferralPolicy {
  min_transfer_quota: number
  bucket_minutes: number
}

export interface ReferralSummary {
  balance: number
  legacy_balance: number
  lifetime_earned: number
  min_transfer_quota: number
  campaign: ReferralCampaign | null
  invited_count: number
  qualified_count: number
  unlocked: boolean
}

export interface ReferralLedger {
  id: number
  campaign_id: number
  source_kind: string
  source_id: string
  entry_kind: string
  amount: number
  source_quota: number
  rate_bps: number
  pool_recipients: number
  pool_index: number
  created_at: number
  campaign_snapshot: string
}

export interface ReferralTransfer {
  id: number
  amount: number
  legacy_part: number
  created_at: number
  source_snapshot: string
}

export interface ReferralRewards {
  summary: ReferralSummary
  entries: ReferralLedger[]
  transfers: ReferralTransfer[]
  entry_total: number
  transfer_total: number
}

export interface ReferralAdminData {
  campaigns: ReferralCampaign[]
  policy: ReferralPolicy
  total_transferred: number
}

export interface ReferralBucket {
  bucket: number
  amount: number
  count: number
}
