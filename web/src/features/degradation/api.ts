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
import { api } from '@/lib/api'

import type { DegradationConfig, DegradationPage } from './types'

type NameListResponse = {
  success: boolean
  message?: string
  data: string[]
}

export async function getDegradationPage(): Promise<DegradationPage> {
  const res = await api.get<DegradationPage>('/api/degradation/')
  return res.data
}

export async function getMonitorGroups(): Promise<NameListResponse> {
  const res = await api.get<NameListResponse>('/api/group/')
  return res.data
}

export async function getMonitorGroupModels(
  group: string
): Promise<NameListResponse> {
  const res = await api.get<NameListResponse>('/api/degradation/models', {
    params: { group },
  })
  return res.data
}

export async function updateDegradationConfig(
  config: DegradationConfig
): Promise<DegradationPage> {
  const res = await api.put<DegradationPage>('/api/degradation/', config)
  return res.data
}

export async function clearDegradationHistory(group: string, model: string) {
  const res = await api.post<{ success: boolean; message?: string }>(
    '/api/degradation/clear',
    { group, model }
  )
  return res.data
}

export async function startDegradationTest(group: string, model: string) {
  const res = await api.post<{
    success: boolean
    message?: string
    data: { task_id: string }
  }>('/api/degradation/test', { group, model })
  return res.data
}

export async function getDegradationTest(taskId: string) {
  const res = await api.get<{
    success: boolean
    message?: string
    data: { status: string; error: string }
  }>(`/api/degradation/test/${encodeURIComponent(taskId)}`)
  return res.data
}
