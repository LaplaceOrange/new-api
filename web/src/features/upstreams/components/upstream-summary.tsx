import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import { formatUpstreamMoney, formatUpstreamTime } from '../lib/format'
import type { Upstream } from '../types'

export function UpstreamStatus(props: { upstream: Upstream }) {
  const { t } = useTranslation()
  const status = props.upstream.forecast.status
  let label = t('Unknown')
  let color = 'text-muted-foreground'
  if (status === 'exhausted') {
    label = t('Exhausted')
    color = 'text-destructive'
  }
  if (status === 'critical') {
    label = t('Critical')
    color = 'text-warning'
  }
  if (status === 'warning') {
    label = t('Warning')
    color = 'text-amber-600 dark:text-amber-400'
  }
  if (status === 'healthy') {
    label = t('Healthy')
    color = 'text-success'
  }
  if (status === 'no_usage') label = t('No usage')
  return <span className={cn('text-sm font-medium', color)}>{label}</span>
}

export function UpstreamSummary(props: {
  upstream: Upstream
  compact?: boolean
}) {
  const { t } = useTranslation()
  const upstream = props.upstream
  const snapshot = upstream.snapshot
  const forecast = upstream.forecast
  return (
    <div className='min-w-0 space-y-3 tracking-normal'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div>
          <p className='text-muted-foreground text-xs'>
            {t('Upstream balance (USD)')}
          </p>
          <p
            className={cn(
              'text-2xl font-semibold tabular-nums',
              forecast.status === 'exhausted' && 'text-destructive',
              forecast.status === 'critical' && 'text-warning'
            )}
          >
            {formatUpstreamMoney(upstream.balance)}
          </p>
        </div>
        <UpstreamStatus upstream={upstream} />
      </div>
      <dl className='grid min-w-0 grid-cols-2 gap-3 text-sm'>
        <div>
          <dt className='text-muted-foreground text-xs'>
            {t('Account spending (USD)')}
          </dt>
          <dd className='tabular-nums'>
            {formatUpstreamMoney(snapshot?.consumption ?? null)}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground text-xs'>
            {t('Estimated remaining days')}
          </dt>
          <dd className='tabular-nums'>
            {forecast.remaining_days === null
              ? '--'
              : t('{{count}} days', {
                  count: Math.round(forecast.remaining_days * 100) / 100,
                })}
          </dd>
        </div>
        {!props.compact && (
          <>
            <div>
              <dt className='text-muted-foreground text-xs'>
                {t('Average daily spending (USD)')}
              </dt>
              <dd className='tabular-nums'>
                {formatUpstreamMoney(forecast.daily_consumption)}
              </dd>
            </div>
            {forecast.suggested_topup !== null &&
              forecast.suggested_topup > 0 && (
                <div>
                  <dt className='text-muted-foreground text-xs'>
                    {t('Suggested top-up (USD)')}
                  </dt>
                  <dd className='tabular-nums'>
                    {formatUpstreamMoney(forecast.suggested_topup)}
                  </dd>
                </div>
              )}
          </>
        )}
      </dl>
      <p className='text-muted-foreground text-xs'>
        {t('Includes all API keys in the upstream account')}
      </p>
      {(forecast.stale || (snapshot && !snapshot.complete)) && (
        <p role='status' className='text-warning text-xs'>
          {t('Stale or incomplete data; estimates may be unavailable')}
        </p>
      )}
      {upstream.credential_blocked && (
        <p role='alert' className='text-destructive text-xs'>
          {t('Credentials unavailable; update credentials to resume refresh')}
        </p>
      )}
      {(upstream.last_error || snapshot?.last_error) && (
        <p role='alert' className='text-destructive text-xs'>
          {t('Upstream refresh failed. Retry or update credentials.')}
        </p>
      )}
      <p className='text-muted-foreground text-xs'>
        {t('Balance updated')}:{' '}
        {formatUpstreamTime(upstream.balance_updated_at, snapshot?.timezone)}
      </p>
      {snapshot && (
        <p className='text-muted-foreground text-xs break-words'>
          {formatUpstreamTime(snapshot.start_at, snapshot.timezone)} -{' '}
          {formatUpstreamTime(snapshot.end_at, snapshot.timezone)} (
          {snapshot.timezone})
        </p>
      )}
    </div>
  )
}
