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

import type { UserCleanupRule, UserCleanupRulesConfig } from './types'

export function createCleanupRule(): UserCleanupRule {
  return {
    id:
      typeof crypto !== 'undefined' && crypto.randomUUID
        ? crypto.randomUUID()
        : `rule-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    condition: 'idle_time',
    action: 'include',
    operator: 'gt',
    duration: 1,
    unit: 'day',
  }
}

export function isIncompleteCleanupRule(rule: UserCleanupRule) {
  if (rule.action !== 'include' && rule.action !== 'ignore') return true
  if (rule.condition === 'group') return !rule.group
  return !rule.operator || !rule.unit || !rule.duration || rule.duration <= 0
}

export function moveCleanupRule(
  rules: UserCleanupRule[],
  index: number,
  direction: -1 | 1
): UserCleanupRule[] {
  const target = index + direction
  if (target < 0 || target >= rules.length) return rules
  const next = [...rules]
  const [rule] = next.splice(index, 1)
  next.splice(target, 0, rule)
  return next
}

export function isActiveCleanupScan(status?: string | null) {
  return status === 'pending' || status === 'running'
}

export function defaultCleanupRulesConfig(): UserCleanupRulesConfig {
  return { rules: [], unmatched_action: 'ignore' }
}
