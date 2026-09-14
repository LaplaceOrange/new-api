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

import { expect, it } from 'vitest'

import {
  createCleanupRule,
  isActiveCleanupScan,
  isIncompleteCleanupRule,
  moveCleanupRule,
} from '../lib'

it('creates a complete idle-time rule by default', () => {
  const rule = createCleanupRule()
  expect(rule.condition).toBe('idle_time')
  expect(rule.action).toBe('include')
  expect(isIncompleteCleanupRule(rule)).toBe(false)
})

it('treats a group rule without a group as incomplete', () => {
  expect(
    isIncompleteCleanupRule({
      id: 'r1',
      condition: 'group',
      action: 'ignore',
    })
  ).toBe(true)
})

it('moves a rule up and ignores out-of-range moves', () => {
  const rules = [createCleanupRule(), createCleanupRule(), createCleanupRule()]
  rules[0].id = 'a'
  rules[1].id = 'b'
  rules[2].id = 'c'
  expect(moveCleanupRule(rules, 2, -1).map((rule) => rule.id)).toEqual([
    'a',
    'c',
    'b',
  ])
  expect(moveCleanupRule(rules, 0, -1)).toBe(rules)
})

it('treats pending and running scans as active', () => {
  expect(isActiveCleanupScan('pending')).toBe(true)
  expect(isActiveCleanupScan('running')).toBe(true)
  expect(isActiveCleanupScan('succeeded')).toBe(false)
})
