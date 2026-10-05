import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, RefreshCw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'

import { getUpstream, refreshUpstreams } from './api'
import { PeriodControls } from './components/period-controls'
import { UpstreamRankings } from './components/upstream-rankings'
import { UpstreamSummary } from './components/upstream-summary'
import type { UpstreamPeriod } from './types'

export function UpstreamDetail(props: { id: number }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [period, setPeriod] = useState<UpstreamPeriod>({
    days: 7,
    mode: 'rolling',
  })
  const query = useQuery({
    queryKey: ['upstreams', props.id, period],
    queryFn: ({ signal }) => getUpstream(props.id, period, signal),
    refetchInterval: (current) =>
      current.state.data?.refreshing ? 15000 : false,
  })
  const refresh = useMutation({
    mutationFn: () => refreshUpstreams(props.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['upstreams'] }),
  })
  return (
    <SectionPageLayout headingLevel='h1' stackActionsOnMobile>
      <SectionPageLayout.Breadcrumb>
        <Link
          to='/upstreams'
          className='text-muted-foreground hover:text-foreground inline-flex items-center gap-2 text-sm'
        >
          <ArrowLeft className='size-4' />
          {t('Upstreams')}
        </Link>
      </SectionPageLayout.Breadcrumb>
      <SectionPageLayout.Title>
        {query.data?.name ?? t('Upstream details')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          disabled={refresh.isPending || query.data?.refreshing}
          onClick={() => refresh.mutate()}
        >
          <RefreshCw className='size-4' />
          {t('Refresh upstream')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='min-w-0 space-y-4 tracking-normal'>
          <PeriodControls value={period} onChange={setPeriod} />
          {query.isPending && <LoadingState />}
          {query.isError && (
            <ErrorState
              title={t('Failed to load upstream')}
              onRetry={() => void query.refetch()}
            />
          )}
          {query.data && (
            <>
              <div className='grid min-w-0 gap-4 md:grid-cols-2'>
                <UpstreamSummary upstream={query.data} />
                <section className='min-w-0 space-y-3'>
                  <h2 className='text-base font-semibold'>
                    {t('Associated channels')}
                  </h2>
                  <div className='flex flex-wrap gap-2'>
                    {query.data.channel_ids.length ? (
                      query.data.channel_ids.map((id) => (
                        <Link
                          key={id}
                          to='/channels'
                          search={{ filter: String(id) }}
                          className='text-sm underline underline-offset-4'
                        >
                          #{id}
                        </Link>
                      ))
                    ) : (
                      <p className='text-muted-foreground text-sm'>
                        {t('No associated channels')}
                      </p>
                    )}
                  </div>
                  <h2 className='text-base font-semibold'>
                    {t('Upstream addresses')}
                  </h2>
                  <ul className='space-y-2 text-sm'>
                    {query.data.addresses.map((address) => (
                      <li key={address} className='break-all'>
                        {address}
                        {address === query.data?.primary_url && (
                          <span className='text-muted-foreground ml-2 text-xs'>
                            {t('Primary')}
                          </span>
                        )}
                      </li>
                    ))}
                  </ul>
                  <p className='text-muted-foreground text-xs'>
                    {query.data.auto_refresh_token
                      ? t('Automatic JWT renewal')
                      : t('Manual JWT renewal')}
                  </p>
                </section>
              </div>
              <UpstreamRankings
                id={props.id}
                period={period}
                refreshing={query.data.refreshing}
              />
            </>
          )}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
