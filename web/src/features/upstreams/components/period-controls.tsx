import { useTranslation } from 'react-i18next'

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import type { UpstreamPeriod } from '../types'

export function PeriodControls(props: {
  value: UpstreamPeriod
  onChange: (value: UpstreamPeriod) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='flex flex-wrap gap-2'>
      <Tabs
        value={String(props.value.days)}
        onValueChange={(days) =>
          props.onChange({
            ...props.value,
            days: Number(days) as UpstreamPeriod['days'],
          })
        }
      >
        <TabsList aria-label={t('Consumption period')}>
          {[1, 7, 30].map((days) => (
            <TabsTrigger key={days} value={String(days)}>
              {t('{{count}} days', { count: days })}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <Tabs
        value={props.value.mode}
        onValueChange={(mode) =>
          props.onChange({
            ...props.value,
            mode: mode as UpstreamPeriod['mode'],
          })
        }
      >
        <TabsList aria-label={t('Period mode')}>
          <TabsTrigger value='rolling'>{t('Rolling')}</TabsTrigger>
          <TabsTrigger value='calendar'>{t('Calendar')}</TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  )
}
