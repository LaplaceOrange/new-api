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
import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, test } from 'vitest'

import {
  TopUpGroupUpgradeRulesEditor,
  type TopUpGroupUpgradeRule,
} from '../topup-group-upgrade-rules-editor'

function RulesEditorHarness({ initialValue }: { initialValue: string }) {
  const [value, setValue] = useState(initialValue)

  return (
    <>
      <TopUpGroupUpgradeRulesEditor
        value={value}
        groupRatio='{"default":1,"vip":2,"svip":3}'
        onChange={setValue}
      />
      <output data-testid='rules-value'>{value}</output>
    </>
  )
}

function savedRules(): TopUpGroupUpgradeRule[] {
  return JSON.parse(screen.getByTestId('rules-value').textContent ?? '[]')
}

describe('top-up group upgrade rules editor', () => {
  test('adds and edits a rule using the configured group list', () => {
    render(<RulesEditorHarness initialValue='[]' />)

    fireEvent.click(screen.getByRole('button', { name: 'Add upgrade rule' }))
    fireEvent.change(screen.getByLabelText('Condition'), {
      target: { value: 'cumulative' },
    })
    fireEvent.change(screen.getByLabelText('Charged amount'), {
      target: { value: '100.00' },
    })
    fireEvent.change(screen.getByLabelText('Target group'), {
      target: { value: 'vip' },
    })

    expect(savedRules()).toEqual([
      { type: 'cumulative', amount: '100.00', group: 'vip' },
    ])
  })

  test('moves rules by priority and removes a rule', () => {
    const initialValue = JSON.stringify([
      { type: 'single', amount: '50.00', group: 'vip' },
      { type: 'cumulative', amount: '200.00', group: 'svip' },
    ])
    render(<RulesEditorHarness initialValue={initialValue} />)

    fireEvent.click(
      screen.getByRole('button', { name: 'Move upgrade rule 2 up' })
    )
    expect(savedRules()).toEqual([
      { type: 'cumulative', amount: '200.00', group: 'svip' },
      { type: 'single', amount: '50.00', group: 'vip' },
    ])

    fireEvent.click(
      screen.getByRole('button', { name: 'Remove upgrade rule 1' })
    )
    expect(savedRules()).toEqual([
      { type: 'single', amount: '50.00', group: 'vip' },
    ])
  })
})
