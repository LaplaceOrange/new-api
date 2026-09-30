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
import { render, screen, within } from '@testing-library/react'
import { expect, it } from 'vitest'

import { StreamTpsCell, TimingMetricsCell } from '../timing-metrics-cell'

it('groups first token, duration and colored average TPS together', () => {
  render(
    <TimingMetricsCell
      useTimeSec={200}
      completionTokens={4000}
      frtMs={35000}
      isStream
    />
  )
  const metrics = screen.getByRole('group', { name: 'Timing' })
  expect(within(metrics).getByText('First token')).toBeVisible()
  expect(within(metrics).getByText('Duration')).toBeVisible()
  expect(within(metrics).getByText('Average TPS')).toBeVisible()
  expect(within(metrics).getByText('35.0s')).toHaveClass('text-orange-700')
  expect(within(metrics).getByText('3m 20s')).toHaveClass('text-orange-700')
  expect(within(metrics).getByText('20 t/s')).toHaveClass('text-amber-700')
})

it('does not duplicate throughput in the request-type cell', () => {
  render(<StreamTpsCell isStream tokensPerSecond={100} />)
  expect(screen.getByText('Stream')).toBeVisible()
  expect(screen.queryByText(/t\/s/)).not.toBeInTheDocument()
})

it('shows unavailable metrics for missing latency and invalid throughput without NaN', () => {
  render(<TimingMetricsCell useTimeSec={0} completionTokens={100} isStream />)
  expect(screen.getAllByText('N/A')).toHaveLength(2)
  expect(screen.queryByText(/NaN|Infinity/)).not.toBeInTheDocument()
})

it('shows unavailable throughput when a finite token count overflows the rate calculation', () => {
  render(
    <TimingMetricsCell
      useTimeSec={Number.MIN_VALUE}
      completionTokens={100}
      isStream={false}
    />
  )
  expect(screen.getByText('N/A')).toBeVisible()
  expect(screen.queryByText(/Infinity/)).not.toBeInTheDocument()
})

it('omits first-token latency for non-streaming requests and throughput for async jobs', () => {
  render(
    <TimingMetricsCell
      useTimeSec={10}
      completionTokens={0}
      isStream={false}
      showThroughput={false}
    />
  )
  expect(screen.getByText('Duration')).toBeVisible()
  expect(screen.queryByText('First token')).not.toBeInTheDocument()
  expect(screen.queryByText('Average TPS')).not.toBeInTheDocument()
})
