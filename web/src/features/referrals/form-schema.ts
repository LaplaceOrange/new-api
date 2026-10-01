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
import { z } from 'zod'

export const referralCampaignSchema = z
  .object({
    id: z.number().int().nonnegative(),
    name: z.string().trim().min(1).max(128),
    enabled: z.boolean(),
    priority: z.number().int().min(-2_147_483_648).max(2_147_483_647),
    start_at: z.number().int().positive(),
    end_at: z.number().int().positive().max(253_402_300_799),
    direction: z.enum(['invitee_to_inviter', 'inviter_to_invitee', 'both']),
    spend_basis: z.enum(['wallet', 'all', 'paid_wallet']),
    required_friends: z.number().int().min(1).max(2_147_483_647),
    paid_threshold_quota: z
      .number()
      .int()
      .nonnegative()
      .max(Number.MAX_SAFE_INTEGER),
    base_bps: z.number().int().min(0).max(10_000),
    inviter_bps: z.number().int().min(0).max(10_000),
    invitee_pool_bps: z.number().int().min(0).max(10_000),
  })
  .refine((campaign) => campaign.end_at > campaign.start_at, {
    path: ['end_at'],
    message: 'End time must be after start time',
  })

export const referralPolicySchema = z.object({
  min_transfer_quota: z.number().int().positive().max(Number.MAX_SAFE_INTEGER),
  bucket_minutes: z
    .number()
    .int()
    .positive()
    .max(Math.floor(2_147_483_647 / 60)),
})
