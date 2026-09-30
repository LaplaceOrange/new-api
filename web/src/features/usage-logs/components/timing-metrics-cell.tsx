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
import { CircleAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatUseTime } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  getFirstResponseTimeColor,
  getResponseTimeColor,
  getThroughputColor,
  type PerformanceVariant,
} from '../lib/format'
import type { LogOtherData } from '../types'

const barColorMap: Record<PerformanceVariant, string> = {
  success: 'bg-emerald-500',
  warning: 'bg-amber-500',
  orange: 'bg-orange-500',
  danger: 'bg-red-500',
  neutral: 'bg-neutral',
}
const textColorMap: Record<PerformanceVariant, string> = {
  success: 'text-emerald-700 dark:text-emerald-400',
  warning: 'text-amber-700 dark:text-amber-400',
  orange: 'text-orange-700 dark:text-orange-400',
  danger: 'text-red-600 dark:text-red-400',
  neutral: 'text-muted-foreground',
}

interface TimingMetricsCellProps {
  useTimeSec: number
  completionTokens: number
  frtMs?: number
  isStream: boolean
  className?: string
  /**
   * `bar` (default) draws a full-height color segment beside the labels,
   * matching the dense desktop table. `dot` swaps that segment for small
   * status dots inline with each label, matching the lighter-weight status
   * indicator used elsewhere on the mobile card.
   */
  indicator?: 'bar' | 'dot'
  compact?: boolean
  showThroughput?: boolean
}

export function TimingMetricsCell(props: TimingMetricsCellProps) {
  const { t } = useTranslation()
  const indicator = props.indicator ?? 'bar'
  const showFirstToken = props.isStream
  const firstTokenSeconds =
    props.frtMs != null && Number.isFinite(props.frtMs) && props.frtMs >= 0
      ? props.frtMs / 1000
      : null
  const firstTokenVariant: PerformanceVariant =
    firstTokenSeconds == null
      ? 'neutral'
      : getFirstResponseTimeColor(firstTokenSeconds)
  const totalTimeVariant = getResponseTimeColor(
    props.useTimeSec,
    props.completionTokens
  )
  const firstTokenLabel =
    firstTokenSeconds == null ? t('N/A') : formatUseTime(firstTokenSeconds)
  const totalTimeLabel =
    totalTimeVariant === 'neutral' ? t('N/A') : formatUseTime(props.useTimeSec)
  const averageTps =
    Number.isFinite(props.useTimeSec) &&
    props.useTimeSec > 0 &&
    Number.isFinite(props.completionTokens) &&
    props.completionTokens > 0
      ? props.completionTokens / props.useTimeSec
      : null
  const tokensPerSecond =
    averageTps != null && Number.isFinite(averageTps) ? averageTps : null
  const throughputVariant =
    tokensPerSecond == null ? 'neutral' : getThroughputColor(tokensPerSecond)
  const showThroughput = props.showThroughput !== false
  const metrics = [
    ...(showFirstToken
      ? [
          {
            label: t('First token'),
            value: firstTokenLabel,
            variant: firstTokenVariant,
          },
        ]
      : []),
    { label: t('Duration'), value: totalTimeLabel, variant: totalTimeVariant },
    ...(showThroughput
      ? [
          {
            label: t('Average TPS'),
            value:
              tokensPerSecond == null
                ? t('N/A')
                : `${Math.round(tokensPerSecond)} t/s`,
            variant: throughputVariant,
          },
        ]
      : []),
  ]

  const labels = (
    <div
      className={cn(
        'flex min-h-8 min-w-0 flex-col justify-center gap-0.5 text-xs leading-tight',
        props.compact &&
          'min-h-0 flex-row flex-wrap items-center gap-x-2.5 gap-y-1'
      )}
    >
      {metrics.map((metric) => (
        <div key={metric.label} className='flex min-w-0 items-baseline gap-1.5'>
          {indicator === 'dot' && (
            <span
              aria-hidden
              className={cn(
                'size-1.5 shrink-0 rounded-full',
                barColorMap[metric.variant]
              )}
            />
          )}
          <span className='text-muted-foreground min-w-0'>{metric.label}</span>
          <span
            className={cn(
              'shrink-0 font-medium tabular-nums',
              textColorMap[metric.variant]
            )}
          >
            {metric.value}
          </span>
        </div>
      ))}
    </div>
  )

  if (indicator === 'dot') {
    return (
      <div
        role='group'
        aria-label={t('Timing')}
        className={cn('flex items-stretch', props.className)}
      >
        {labels}
      </div>
    )
  }

  return (
    <div
      role='group'
      aria-label={t('Timing')}
      className={cn('flex items-stretch gap-2', props.className)}
    >
      <span
        aria-hidden
        className={cn(
          'flex w-1 shrink-0 flex-col overflow-hidden rounded-full',
          !showFirstToken && !showThroughput && barColorMap[totalTimeVariant]
        )}
      >
        {(showFirstToken || showThroughput) &&
          metrics.map((metric) => (
            <span
              key={metric.label}
              className={cn('flex-1', barColorMap[metric.variant])}
            />
          ))}
      </span>
      {labels}
    </div>
  )
}

interface StreamTpsCellProps {
  isStream: boolean
  compact?: boolean
  /** Task logs are asynchronous jobs; stream vs non-stream does not apply. */
  isTask?: boolean
  tokensPerSecond?: number | null
  streamStatus?: LogOtherData['stream_status']
  className?: string
}

export function StreamTpsCell(props: StreamTpsCellProps) {
  const { t } = useTranslation()
  const showStreamError =
    props.isStream && props.streamStatus && props.streamStatus.status !== 'ok'
  let streamLabel = props.isStream ? t('Stream') : t('Non-stream')
  if (props.isTask) {
    streamLabel = t('Async')
  }

  return (
    <div
      className={cn(
        'flex shrink-0 flex-col items-start justify-center gap-0.5 text-xs leading-tight',
        props.compact && 'flex-row flex-wrap items-center gap-1.5',
        props.className
      )}
    >
      <span
        className={cn(
          'inline-flex items-center gap-1 font-medium',
          props.isStream ? 'text-info' : 'text-muted-foreground'
        )}
      >
        {streamLabel}
        {showStreamError && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger
                render={<CircleAlert className='text-destructive size-3' />}
              />
              <TooltipContent>
                <div className='space-y-0.5 text-xs'>
                  <p>
                    {t('Stream Status')}: {t('Error')}
                  </p>
                  <p>{props.streamStatus?.end_reason || 'unknown'}</p>
                  {(props.streamStatus?.error_count ?? 0) > 0 && (
                    <p>
                      {t('Soft Errors')}: {props.streamStatus?.error_count}
                    </p>
                  )}
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}
      </span>
    </div>
  )
}
