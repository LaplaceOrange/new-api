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

import type { User, GetUsersResponse } from '@/features/users/types'
import { api } from '@/lib/api'

import type {
  UserCleanupRecord,
  UserCleanupRulesConfig,
  UserCleanupScanTask,
} from './types'

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

export async function getUserCleanupRules(): Promise<
  ApiResponse<UserCleanupRulesConfig>
> {
  const res = await api.get('/api/user/cleanup/rules')
  return res.data
}

export async function saveUserCleanupRules(
  config: UserCleanupRulesConfig
): Promise<ApiResponse<UserCleanupRulesConfig>> {
  const res = await api.put('/api/user/cleanup/rules', config)
  return res.data
}

export async function startUserCleanupScan(): Promise<
  ApiResponse<UserCleanupScanTask>
> {
  const res = await api.post('/api/user/cleanup/scan')
  return res.data
}

export async function getUserCleanupScan(): Promise<
  ApiResponse<UserCleanupScanTask | null>
> {
  const res = await api.get('/api/user/cleanup/scan')
  return res.data
}

export async function getUserCleanupCandidates(params: {
  p?: number
  page_size?: number
}): Promise<GetUsersResponse> {
  const res = await api.get('/api/user/cleanup/candidates', {
    params: { p: params.p ?? 1, page_size: params.page_size ?? 20 },
  })
  return res.data
}

export async function getUserCleanupCandidateIds(): Promise<
  ApiResponse<number[]>
> {
  const res = await api.get('/api/user/cleanup/candidates/ids')
  return res.data
}

export async function applyUserCleanup(
  userIds: number[]
): Promise<ApiResponse<{ disabled_count: number }>> {
  const res = await api.post('/api/user/cleanup/apply', { user_ids: userIds })
  return res.data
}

export async function getUserCleanupHistory(params: {
  p?: number
  page_size?: number
}): Promise<
  ApiResponse<{
    items: UserCleanupRecord[]
    total: number
    page: number
    page_size: number
  }>
> {
  const res = await api.get('/api/user/cleanup/history', {
    params: { p: params.p ?? 1, page_size: params.page_size ?? 20 },
  })
  return res.data
}

export type { User }
