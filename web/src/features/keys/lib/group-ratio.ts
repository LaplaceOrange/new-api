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

export type GroupRatioValue = number | string | null | undefined

function formatRatioValue(value: number | string): string {
  if (typeof value === 'number') {
    return String(Number(value.toFixed(10)))
  }
  return value
}

export function formatGroupRatio(
  ratio: GroupRatioValue,
  topupRatio: GroupRatioValue,
  ratioLabel: string,
  topupRatioLabel: string
): string | null {
  if (ratio === undefined || ratio === null || ratio === '') return null
  if (topupRatio === undefined || topupRatio === null || topupRatio === '') {
    return `${formatRatioValue(ratio)}x ${ratioLabel}`
  }

  const numericRatio = Number(ratio)
  const numericTopupRatio = Number(topupRatio)
  if (!Number.isFinite(numericRatio) || !Number.isFinite(numericTopupRatio)) {
    return `${formatRatioValue(ratio)}x ${ratioLabel}`
  }

  const effectiveRatio = Number((numericRatio * numericTopupRatio).toFixed(10))
  return `${formatRatioValue(numericRatio)}${ratioLabel}*${formatRatioValue(numericTopupRatio)}${topupRatioLabel}=${formatRatioValue(effectiveRatio)}${ratioLabel}`
}
