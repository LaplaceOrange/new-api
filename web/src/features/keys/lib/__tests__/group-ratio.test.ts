/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import { formatGroupRatio } from '../group-ratio'

describe('group ratio display', () => {
  test('shows the effective ratio after applying the top-up ratio', () => {
    expect(formatGroupRatio(1.8, 0.1, '倍率', '充值倍率')).toBe(
      '1.8倍率*0.1充值倍率=0.18倍率'
    )
  })

  test('keeps the legacy display when no top-up ratio is available', () => {
    expect(formatGroupRatio(1.8, undefined, 'Ratio', 'Top-up ratio')).toBe(
      '1.8x Ratio'
    )
  })
})
