import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  StaticDataTable,
  type StaticDataTableColumn,
} from '@/components/data-table'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { getUpstreamRankings } from '../api'
import { formatUpstreamTime } from '../lib/format'
import type {
  RankingOptions,
  UpstreamPeriod,
  UpstreamRankings as Rankings,
} from '../types'

export function UpstreamRankings(props: {
  id: number
  period: UpstreamPeriod
  refreshing: boolean
}) {
  const { t } = useTranslation()
  const [dimension, setDimension] =
    useState<RankingOptions['dimension']>('users')
  const [metric, setMetric] = useState<RankingOptions['metric']>('quota')
  const [limit, setLimit] = useState<RankingOptions['limit']>(20)
  const options = { ...props.period, dimension, metric, limit }
  const query = useQuery({
    queryKey: ['upstreams', props.id, 'rankings', options],
    queryFn: ({ signal }) => getUpstreamRankings(props.id, options, signal),
    refetchInterval: props.refreshing ? 15000 : false,
  })
  const columns = useMemo<StaticDataTableColumn<Rankings['items'][number]>[]>(
    () => [
      {
        id: 'name',
        header: dimension === 'users' ? t('User') : t('Model'),
        cell: (item) => (
          <span className='block max-w-96 break-words' title={item.name}>
            {item.name}
            {item.user_id !== undefined && (
              <span className='text-muted-foreground ml-2 text-xs'>
                #{item.user_id}
              </span>
            )}
          </span>
        ),
      },
      {
        id: 'quota',
        header: t('Local channel quota'),
        cell: (item) => (
          <span className='tabular-nums'>{item.quota.toLocaleString()}</span>
        ),
        className: 'text-right',
      },
      {
        id: 'requests',
        header: t('Requests'),
        cell: (item) => (
          <span className='tabular-nums'>{item.requests.toLocaleString()}</span>
        ),
        className: 'text-right',
      },
      {
        id: 'tokens',
        header: t('Tokens'),
        cell: (item) => (
          <span className='tabular-nums'>{item.tokens.toLocaleString()}</span>
        ),
        className: 'text-right',
      },
    ],
    [dimension, t]
  )
  return (
    <section className='min-w-0 space-y-3 border-t pt-4'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <h2 className='text-base font-semibold'>
          {t('Local channel usage ranking')}
        </h2>
        <div className='flex flex-wrap gap-2'>
          <Tabs
            value={dimension}
            onValueChange={(value) =>
              setDimension(value as RankingOptions['dimension'])
            }
          >
            <TabsList aria-label={t('Ranking dimension')}>
              <TabsTrigger value='users'>{t('Users')}</TabsTrigger>
              <TabsTrigger value='models'>{t('Models')}</TabsTrigger>
            </TabsList>
          </Tabs>
          <Select
            value={metric}
            onValueChange={(value) => {
              if (value) setMetric(value as RankingOptions['metric'])
            }}
          >
            <SelectTrigger aria-label={t('Ranking metric')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='quota'>{t('Quota')}</SelectItem>
              <SelectItem value='requests'>{t('Requests')}</SelectItem>
              <SelectItem value='tokens'>{t('Tokens')}</SelectItem>
            </SelectContent>
          </Select>
          <Select
            value={limit}
            onValueChange={(value) => {
              if (value) setLimit(Number(value) as RankingOptions['limit'])
            }}
          >
            <SelectTrigger aria-label={t('Ranking limit')}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {[10, 20, 50].map((value) => (
                <SelectItem key={value} value={value}>
                  {value}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <p className='text-muted-foreground text-xs'>
        {t('Local channel quota is not upstream account spending')}
      </p>
      {query.isPending && <LoadingState />}
      {query.isError && (
        <ErrorState
          title={t('Failed to load rankings')}
          onRetry={() => void query.refetch()}
        />
      )}
      {query.data && (
        <>
          <p className='text-muted-foreground text-xs break-words'>
            {formatUpstreamTime(query.data.start_at, query.data.timezone)} -{' '}
            {formatUpstreamTime(query.data.end_at, query.data.timezone)} (
            {query.data.timezone})
          </p>
          {query.data.items.length ? (
            <StaticDataTable
              data={query.data.items}
              columns={columns}
              getRowKey={(item) => item.user_id ?? item.name}
            />
          ) : (
            <EmptyState title={t('No local usage in this period')} />
          )}
        </>
      )}
    </section>
  )
}
