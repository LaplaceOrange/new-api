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

type CleanupPage<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
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

export function getAllUserCleanupCandidates(): Promise<GetUsersResponse> {
  return fetchAllCleanupPages((page) =>
    getUserCleanupCandidates({ p: page, page_size: 100 })
  )
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
}): Promise<ApiResponse<CleanupPage<UserCleanupRecord>>> {
  const res = await api.get('/api/user/cleanup/history', {
    params: { p: params.p ?? 1, page_size: params.page_size ?? 20 },
  })
  return res.data
}

export function getAllUserCleanupHistory(): Promise<
  ApiResponse<CleanupPage<UserCleanupRecord>>
> {
  return fetchAllCleanupPages((page) =>
    getUserCleanupHistory({ p: page, page_size: 100 })
  )
}

async function fetchAllCleanupPages<T>(
  getPage: (page: number) => Promise<ApiResponse<CleanupPage<T>>>
): Promise<ApiResponse<CleanupPage<T>>> {
  const firstPage = await getPage(1)
  if (!firstPage.success || !firstPage.data) return firstPage

  const pageSize = firstPage.data.page_size || 100
  const pageCount = Math.ceil(
    Math.max(firstPage.data.total, firstPage.data.items.length) / pageSize
  )
  if (pageCount <= 1) return firstPage

  const items = [...firstPage.data.items]
  for (let startPage = 2; startPage <= pageCount; startPage += 10) {
    const batchSize = Math.min(10, pageCount - startPage + 1)
    const pages = await Promise.all(
      Array.from({ length: batchSize }, (_, index) =>
        getPage(startPage + index)
      )
    )
    const failedPage = pages.find((page) => !page.success || !page.data)
    if (failedPage) return failedPage
    items.push(...pages.flatMap((page) => page.data?.items ?? []))
  }

  return { ...firstPage, data: { ...firstPage.data, items } }
}

export type { User }
