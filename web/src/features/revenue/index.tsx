import { useNavigate, useSearch } from '@tanstack/react-router'
import { VChart } from '@visactor/react-vchart'
import {
  BarChart3,
  CreditCard,
  DollarSign,
  ShoppingBag,
  Users,
} from 'lucide-react'
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'
import { formatLocalCurrencyAmount } from '@/lib/currency'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'

import { useRevenue } from './hooks/use-revenue'
import type { RevenuePeriod, RevenueProvider, RevenueSnapshot } from './types'

const PERIODS: Array<{ id: RevenuePeriod; labelKey: string }> = [
  { id: 'today', labelKey: 'Today' },
  { id: 'week', labelKey: 'Week' },
  { id: 'month', labelKey: 'Month' },
  { id: 'year', labelKey: 'Year' },
  { id: 'all', labelKey: 'All time' },
]

const VALID_PERIODS = new Set<RevenuePeriod>([
  'today',
  'week',
  'month',
  'year',
  'all',
])

export function Revenue() {
  const { t } = useTranslation()
  const search = useSearch({ from: '/_authenticated/revenue/' })
  const navigate = useNavigate()
  const period = VALID_PERIODS.has(search.period as RevenuePeriod)
    ? (search.period as RevenuePeriod)
    : 'week'
  const revenueQuery = useRevenue(period)
  const snapshot = revenueQuery.data?.data

  let content: ReactNode
  if (revenueQuery.isLoading) {
    content = <RevenueLoading />
  } else if (!snapshot) {
    content = (
      <div className='rounded-xl border border-dashed px-6 py-12 text-center'>
        <h2 className='font-semibold'>{t('Unable to load revenue')}</h2>
        <p className='text-muted-foreground mt-2 text-sm'>
          {revenueQuery.error instanceof Error
            ? revenueQuery.error.message
            : t('Unable to load revenue data')}
        </p>
      </div>
    )
  } else {
    content = <RevenueContent snapshot={snapshot} />
  }

  const handlePeriodChange = (value: string) => {
    if (!VALID_PERIODS.has(value as RevenuePeriod)) return
    navigate({
      to: '/revenue',
      search: (previous) => ({ ...previous, period: value as RevenuePeriod }),
    })
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        <span className='inline-flex min-w-0 items-center gap-2'>
          <span className='truncate'>{t('Revenue')}</span>
          <Badge variant='outline' className='shrink-0'>
            {t('Admin')}
          </Badge>
        </span>
      </SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <PageTransition className='space-y-4'>
          <div className='bg-card flex flex-col gap-3 rounded-xl border px-4 py-4 sm:flex-row sm:items-center sm:justify-between'>
            <div>
              <h1 className='text-lg font-semibold tracking-tight'>
                {t('Payment revenue')}
              </h1>
              <p className='text-muted-foreground mt-1 text-sm'>
                {t('Online payment performance across the selected period.')}
              </p>
            </div>
            <Tabs value={period} onValueChange={handlePeriodChange}>
              <TabsList className='w-full overflow-x-auto sm:w-auto'>
                {PERIODS.map((item) => (
                  <TabsTrigger key={item.id} value={item.id}>
                    {t(item.labelKey)}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>

          {content}
        </PageTransition>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function RevenueContent(props: { snapshot: RevenueSnapshot }) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const chartTextColor =
    resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.68)'
      : 'rgba(15, 23, 42, 0.58)'
  const chartGridColor =
    resolvedTheme === 'dark'
      ? 'rgba(255, 255, 255, 0.12)'
      : 'rgba(15, 23, 42, 0.12)'

  const chartSpec = useMemo(() => {
    if (props.snapshot.trend.length === 0) return null
    return {
      type: 'line' as const,
      data: [{ id: 'revenue-trend', values: props.snapshot.trend }],
      xField: 'label',
      yField: 'revenue',
      smooth: true,
      line: { style: { stroke: '#2563eb', lineWidth: 2.5 } },
      point: { visible: true, style: { size: 4, fill: '#2563eb' } },
      legends: { visible: false },
      axes: [
        {
          orient: 'bottom',
          label: {
            style: { fill: chartTextColor, fontSize: 10 },
            autoHide: true,
            autoLimit: true,
          },
          tick: { visible: false },
        },
        {
          orient: 'left',
          label: { style: { fill: chartTextColor, fontSize: 10 } },
          grid: {
            visible: true,
            style: { lineDash: [3, 3], stroke: chartGridColor },
          },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: () => t('Revenue'),
              value: (datum: Record<string, unknown>) =>
                formatLocalCurrencyAmount(Number(datum?.revenue) || 0),
            },
          ],
        },
      },
      animationAppear: { duration: 500 },
    }
  }, [chartGridColor, chartTextColor, props.snapshot.trend, t])

  return (
    <div className='space-y-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        <RevenueMetric
          icon={<DollarSign />}
          label={t('Revenue')}
          value={formatLocalCurrencyAmount(props.snapshot.summary.revenue)}
          change={props.snapshot.summary.revenue_growth_pct}
        />
        <RevenueMetric
          icon={<ShoppingBag />}
          label={t('Orders')}
          value={props.snapshot.summary.orders.toLocaleString()}
          change={props.snapshot.summary.order_growth_pct}
        />
        <RevenueMetric
          icon={<Users />}
          label={t('Customers')}
          value={props.snapshot.summary.customers.toLocaleString()}
        />
        <RevenueMetric
          icon={<CreditCard />}
          label={t('Average order')}
          value={formatLocalCurrencyAmount(
            props.snapshot.summary.average_order
          )}
        />
      </div>

      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.6fr)_minmax(320px,1fr)]'>
        <Card>
          <CardHeader>
            <CardTitle className='inline-flex items-center gap-2'>
              <BarChart3 className='text-primary size-4' aria-hidden='true' />
              {t('Revenue trend')}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className='h-72'>
              {themeReady && chartSpec ? (
                <VChart
                  key={`revenue-trend-${resolvedTheme}`}
                  spec={{
                    ...chartSpec,
                    theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                    background: 'transparent',
                  }}
                  option={VCHART_OPTION}
                />
              ) : (
                <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                  {t('No revenue history available')}
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <ProviderBreakdown providers={props.snapshot.providers} />
      </div>

      <div className='grid gap-4 xl:grid-cols-2'>
        <CustomerRankingCard
          customers={props.snapshot.customers}
          title={t('Top customers by payment amount')}
          mode='amount'
        />
        <CustomerRankingCard
          customers={props.snapshot.customer_order_rankings}
          title={t('Top customers by payment count')}
          mode='count'
        />
      </div>
    </div>
  )
}

function RevenueMetric(props: {
  icon: React.ReactNode
  label: string
  value: string
  change?: number
}) {
  const { t } = useTranslation()
  const change = props.change
  return (
    <Card size='sm'>
      <CardContent className='flex items-center gap-3'>
        <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
          {props.icon}
        </span>
        <div className='min-w-0'>
          <p className='text-muted-foreground text-xs'>{props.label}</p>
          <p className='mt-0.5 truncate text-xl font-semibold tabular-nums'>
            {props.value}
          </p>
          {change != null && (
            <p className='text-muted-foreground mt-0.5 text-[11px] tabular-nums'>
              {change >= 0 ? '+' : ''}
              {change.toFixed(1)}% {t('vs. prior')}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

function ProviderBreakdown(props: { providers: RevenueProvider[] }) {
  const { t } = useTranslation()
  const maxRevenue = Math.max(...props.providers.map((item) => item.revenue), 1)
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('Payment channels')}</CardTitle>
      </CardHeader>
      <CardContent>
        {props.providers.length === 0 ? (
          <p className='text-muted-foreground py-8 text-center text-sm'>
            {t('No payment data available')}
          </p>
        ) : (
          <div className='space-y-4'>
            {props.providers.map((provider) => (
              <div key={provider.provider} className='space-y-1.5'>
                <div className='flex items-center justify-between gap-3 text-sm'>
                  <span className='truncate font-medium'>
                    {provider.provider}
                  </span>
                  <span className='text-muted-foreground shrink-0 tabular-nums'>
                    {formatLocalCurrencyAmount(provider.revenue)}
                  </span>
                </div>
                <div className='bg-muted h-2 overflow-hidden rounded-full'>
                  <div
                    className='bg-primary h-full rounded-full transition-[width]'
                    style={{
                      width: `${(provider.revenue / maxRevenue) * 100}%`,
                    }}
                  />
                </div>
                <div className='text-muted-foreground flex justify-between text-xs tabular-nums'>
                  <span>
                    {provider.orders.toLocaleString()} {t('orders')}
                  </span>
                  <span>{(provider.share * 100).toFixed(1)}%</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function CustomerRankingCard(props: {
  customers: RevenueSnapshot['customers']
  title: string
  mode: 'amount' | 'count'
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{props.title}</CardTitle>
      </CardHeader>
      <CardContent>
        {props.customers.length === 0 ? (
          <p className='text-muted-foreground py-8 text-center text-sm'>
            {t('No customer payment data available')}
          </p>
        ) : (
          <div className='divide-border/60 divide-y'>
            {props.customers.map((customer) => {
              let primaryValue: string
              let secondaryValue: string
              if (props.mode === 'amount') {
                primaryValue = formatLocalCurrencyAmount(customer.revenue)
                secondaryValue = `${customer.orders.toLocaleString()} ${t('payments')}`
              } else {
                primaryValue = `${customer.orders.toLocaleString()} ${t('payments')}`
                secondaryValue = formatLocalCurrencyAmount(customer.revenue)
              }

              return (
                <div
                  key={customer.user_id}
                  className='flex items-center gap-3 py-3 first:pt-0 last:pb-0'
                >
                  <span className='text-muted-foreground w-6 text-right font-mono text-xs tabular-nums'>
                    {customer.rank}.
                  </span>
                  <Avatar className='size-8 shrink-0'>
                    <AvatarFallback
                      style={getUserAvatarStyle(
                        customer.display_name || customer.username
                      )}
                    >
                      {getUserAvatarFallback(
                        customer.display_name || customer.username
                      )}
                    </AvatarFallback>
                  </Avatar>
                  <div className='min-w-0 flex-1'>
                    <div className='flex items-center gap-1.5'>
                      <span className='truncate text-sm font-medium'>
                        {customer.username}
                      </span>
                      <span className='text-muted-foreground/70 font-mono text-[10px] tabular-nums'>
                        #{customer.user_id}
                      </span>
                    </div>
                    {customer.display_name && (
                      <p className='text-muted-foreground truncate text-xs'>
                        {customer.display_name}
                      </p>
                    )}
                  </div>
                  <div className='shrink-0 text-right'>
                    <p className='text-sm font-semibold tabular-nums'>
                      {primaryValue}
                    </p>
                    <p className='text-muted-foreground text-xs tabular-nums'>
                      {secondaryValue}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function RevenueLoading() {
  return (
    <div className='space-y-4'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        {Array.from({ length: 4 }, (_, index) => (
          <Skeleton key={index} className='h-24 rounded-xl' />
        ))}
      </div>
      <div className='grid gap-4 xl:grid-cols-[minmax(0,1.6fr)_minmax(320px,1fr)]'>
        <Skeleton className='h-[360px] rounded-xl' />
        <Skeleton className='h-[360px] rounded-xl' />
      </div>
      <Skeleton className='h-64 rounded-xl' />
    </div>
  )
}
