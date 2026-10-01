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
import { getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePagination, StaticDataTable } from '@/components/data-table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatQuota } from '@/lib/format'

import type { ReferralLedger, ReferralRewards, ReferralTransfer } from './types'

function campaignName(entry: ReferralLedger): string {
  try {
    const snapshot: unknown = JSON.parse(entry.campaign_snapshot)
    if (
      snapshot &&
      typeof snapshot === 'object' &&
      'name' in snapshot &&
      typeof snapshot.name === 'string'
    ) {
      return snapshot.name
    }
  } catch {
    /* Older entries may have no snapshot. */
  }
  return String(entry.campaign_id)
}

function transferCampaigns(transfer: ReferralTransfer): string {
  try {
    const sources: unknown = JSON.parse(transfer.source_snapshot)
    if (Array.isArray(sources)) {
      const ids = sources.flatMap((source: unknown) => {
        if (
          source &&
          typeof source === 'object' &&
          'campaign_id' in source &&
          typeof source.campaign_id === 'number'
        ) {
          return [source.campaign_id]
        }
        return []
      })
      return [...new Set(ids)].join(', ')
    }
  } catch {
    /* Legacy transfers have no campaign IDs. */
  }
  return ''
}

export function ReferralRewardHistory(props: {
  rewards: ReferralRewards
  page: number
  pending: boolean
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const [tab, setTab] = useState('rewards')
  const table = useReactTable({
    data: props.rewards.entries,
    columns: [],
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    rowCount:
      tab === 'rewards'
        ? props.rewards.entry_total
        : props.rewards.transfer_total,
    state: { pagination: { pageIndex: props.page - 1, pageSize: 20 } },
    onPaginationChange: (updater) => {
      if (props.pending) return
      const previous = { pageIndex: props.page - 1, pageSize: 20 }
      const next = typeof updater === 'function' ? updater(previous) : updater
      props.onPageChange(next.pageIndex + 1)
    },
  })
  return (
    <Tabs
      value={tab}
      onValueChange={(value) => {
        setTab(value)
        props.onPageChange(1)
      }}
      className='min-w-0'
    >
      <TabsList>
        <TabsTrigger value='rewards'>{t('Rebate sources')}</TabsTrigger>
        <TabsTrigger value='transfers'>{t('Withdrawal history')}</TabsTrigger>
      </TabsList>
      <TabsContent value='rewards'>
        <StaticDataTable
          data={props.rewards.entries ?? []}
          getRowKey={(entry) => entry.id}
          emptyContent={t('No referral rewards')}
          columns={[
            { id: 'campaign', header: t('Campaign'), cell: campaignName },
            {
              id: 'source',
              header: t('Source'),
              cell: (entry) => (
                <div className='min-w-0'>
                  <span>
                    {entry.source_kind === 'wallet'
                      ? t('Wallet')
                      : t('Subscription')}
                  </span>
                  <div
                    className='max-w-64 truncate font-mono text-xs'
                    title={entry.source_id}
                  >
                    {entry.source_id}
                  </div>
                </div>
              ),
            },
            {
              id: 'type',
              header: t('Type'),
              cell: (entry) => {
                if (entry.amount < 0) return t('Refund')
                return entry.entry_kind === 'inviter'
                  ? t('To inviter')
                  : t('To invited friends')
              },
            },
            {
              id: 'amount',
              header: t('Amount'),
              cell: (entry) => formatQuota(entry.amount),
            },
            {
              id: 'rate',
              header: t('Rebate (%)'),
              cell: (entry) =>
                entry.pool_recipients > 0
                  ? `${entry.rate_bps / 100}% / ${entry.pool_recipients}`
                  : `${entry.rate_bps / 100}%`,
            },
            {
              id: 'time',
              header: t('Time'),
              cell: (entry) =>
                new Date(entry.created_at * 1000).toLocaleString(),
            },
          ]}
        />
      </TabsContent>
      <TabsContent value='transfers'>
        <StaticDataTable
          data={props.rewards.transfers ?? []}
          getRowKey={(entry) => entry.id}
          emptyContent={t('No withdrawals')}
          columns={[
            {
              id: 'amount',
              header: t('Amount'),
              cell: (entry) => formatQuota(entry.amount),
            },
            {
              id: 'legacy',
              header: t('Legacy rewards'),
              cell: (entry) => formatQuota(entry.legacy_part),
            },
            { id: 'campaign', header: t('Campaign'), cell: transferCampaigns },
            {
              id: 'time',
              header: t('Time'),
              cell: (entry) =>
                new Date(entry.created_at * 1000).toLocaleString(),
            },
          ]}
        />
      </TabsContent>
      <fieldset disabled={props.pending}>
        <DataTablePagination table={table} compact />
      </fieldset>
    </Tabs>
  )
}
