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

export type UserCleanupCondition = 'group' | 'idle_time'
export type UserCleanupAction = 'include' | 'ignore'
export type UserCleanupOperator = 'gt' | 'gte' | 'eq' | 'lt' | 'lte'
export type UserCleanupUnit = 'hour' | 'day' | 'month'

export type UserCleanupRule = {
  id: string
  condition: UserCleanupCondition
  action: UserCleanupAction
  group?: string
  operator?: UserCleanupOperator
  duration?: number
  unit?: UserCleanupUnit
}

export type UserCleanupRulesConfig = {
  rules: UserCleanupRule[]
  unmatched_action: UserCleanupAction
}

export type UserCleanupScanState = {
  total?: number
  processed?: number
  matched?: number
  progress?: number
  skipped_rule_count?: number
  scan_id?: string
}

export type UserCleanupScanTask = {
  task_id: string
  type: string
  status: 'pending' | 'running' | 'succeeded' | 'failed' | string
  state?: UserCleanupScanState | null
  result?: {
    matched?: number
    skipped_rule_count?: number
    scan_id?: string
  } | null
  error?: string
}

export type UserCleanupRecord = {
  id: number
  user_id: number
  username: string
  display_name: string
  status: number
  group: string
  quota: number
  cleaned_at: number
}
