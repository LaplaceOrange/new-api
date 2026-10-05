import { Link } from '@tanstack/react-router'
import type { ColumnDef } from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePage, useDataTable } from '@/components/data-table'

import { formatUpstreamMoney } from '../lib/format'
import type { Upstream } from '../types'
import { UpstreamActions } from './upstream-actions'
import { UpstreamStatus, UpstreamSummary } from './upstream-summary'

type Props = {
  items: Upstream[]
  loading: boolean
  fetching: boolean
  root: boolean
  refreshPending: boolean
  onRefresh: (id: number) => void
  onEdit: (upstream: Upstream) => void
  onDelete: (upstream: Upstream) => void
}

export function UpstreamList(props: Props) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const columns = useMemo<ColumnDef<Upstream, unknown>[]>(
    () => [
      {
        accessorKey: 'name',
        header: t('Name'),
        cell: ({ row }) => (
          <Link
            to='/upstreams/$id'
            params={{ id: String(row.original.id) }}
            title={row.original.name}
            className='block max-w-72 text-base font-semibold break-words hover:underline'
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: 'primary_url',
        header: t('Primary address'),
        cell: ({ row }) => (
          <span
            className='block max-w-72 truncate'
            title={row.original.primary_url}
          >
            {row.original.primary_url}
          </span>
        ),
      },
      {
        accessorKey: 'balance',
        header: t('Upstream balance (USD)'),
        cell: ({ row }) => (
          <span className='text-2xl font-semibold tabular-nums'>
            {formatUpstreamMoney(row.original.balance)}
          </span>
        ),
      },
      {
        id: 'spending',
        header: t('Account spending (USD)'),
        cell: ({ row }) => (
          <div>
            <span className='tabular-nums'>
              {formatUpstreamMoney(row.original.snapshot?.consumption ?? null)}
            </span>
            {(row.original.forecast.stale ||
              !row.original.snapshot?.complete) && (
              <p className='text-warning text-xs'>{t('Stale or incomplete')}</p>
            )}
          </div>
        ),
      },
      {
        id: 'status',
        header: t('Status'),
        cell: ({ row }) => <UpstreamStatus upstream={row.original} />,
      },
      {
        id: 'actions',
        header: t('Actions'),
        cell: ({ row }) => (
          <UpstreamActions
            upstream={row.original}
            root={props.root}
            pending={props.refreshPending}
            onRefresh={props.onRefresh}
            onEdit={props.onEdit}
            onDelete={props.onDelete}
          />
        ),
      },
    ],
    [
      t,
      props.root,
      props.refreshPending,
      props.onRefresh,
      props.onEdit,
      props.onDelete,
    ]
  )
  const { table } = useDataTable({
    data: props.items,
    columns,
    globalFilter: search,
    onGlobalFilterChange: setSearch,
    enableRowSelection: false,
    getRowId: (row) => String(row.id),
    globalFilterFn: (row, _columnId, value) =>
      `${row.original.name} ${row.original.addresses.join(' ')}`
        .toLowerCase()
        .includes(String(value).toLowerCase()),
  })
  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={props.loading}
      isFetching={props.fetching}
      fixedHeight={false}
      enableCardView
      viewModeStorageKey='upstreams:view-mode'
      defaultViewMode='card'
      toolbarProps={{ searchPlaceholder: t('Search upstreams...') }}
      emptyTitle={t('No upstreams found')}
      getColumnClassName={(id) =>
        ['balance', 'spending'].includes(id) ? 'text-right' : undefined
      }
      renderCard={(row) => (
        <article className='min-w-0 space-y-3 rounded-lg border p-4'>
          <div className='flex items-start justify-between gap-2'>
            <Link
              to='/upstreams/$id'
              params={{ id: String(row.original.id) }}
              title={row.original.name}
              className='min-w-0 text-base font-semibold break-words hover:underline'
            >
              {row.original.name}
            </Link>
            <UpstreamActions
              upstream={row.original}
              root={props.root}
              pending={props.refreshPending}
              onRefresh={props.onRefresh}
              onEdit={props.onEdit}
              onDelete={props.onDelete}
            />
          </div>
          <p
            className='text-muted-foreground truncate text-xs'
            title={row.original.primary_url}
          >
            {row.original.primary_url}
          </p>
          <UpstreamSummary upstream={row.original} compact />
        </article>
      )}
    />
  )
}
