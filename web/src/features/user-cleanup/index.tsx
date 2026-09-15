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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type {
  ColumnDef,
  PaginationState,
  RowSelectionState,
} from '@tanstack/react-table'
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { DataTablePage, useDataTable } from '@/components/data-table'
import { GroupBadge } from '@/components/group-badge'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Input } from '@/components/ui/input'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getGroups } from '@/features/users/api'
import { useUsersColumns } from '@/features/users/components/users-columns'
import { USER_STATUSES } from '@/features/users/constants'
import { formatQuota, formatTimestamp } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  createServerError,
  requireServerSuccess,
} from '@/lib/server-error-message'

import {
  applyUserCleanup,
  getUserCleanupCandidateIds,
  getUserCleanupCandidates,
  getUserCleanupHistory,
  getUserCleanupRules,
  getUserCleanupScan,
  saveUserCleanupRules,
  startUserCleanupScan,
} from './api'
import {
  createCleanupRule,
  defaultCleanupRulesConfig,
  isActiveCleanupScan,
  isIncompleteCleanupRule,
  moveCleanupRule,
} from './lib'
import type {
  UserCleanupAction,
  UserCleanupCondition,
  UserCleanupOperator,
  UserCleanupRecord,
  UserCleanupRule,
  UserCleanupRulesConfig,
  UserCleanupUnit,
} from './types'

const RULES_QUERY = ['user-cleanup-rules'] as const
const SCAN_QUERY = ['user-cleanup-scan'] as const
const CANDIDATES_QUERY = ['user-cleanup-candidates'] as const
const CANDIDATE_IDS_QUERY = ['user-cleanup-candidate-ids'] as const
const HISTORY_QUERY = ['user-cleanup-history'] as const

export function UserCleanupPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [rulesConfig, setRulesConfig] = useState<UserCleanupRulesConfig>(
    defaultCleanupRulesConfig()
  )
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [candidatePage, setCandidatePage] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [historyPage, setHistoryPage] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({})
  const saveTimer = useRef<number | undefined>(undefined)
  const skipNextSave = useRef(true)

  const rulesQuery = useQuery({
    queryKey: RULES_QUERY,
    queryFn: async () => requireServerSuccess(await getUserCleanupRules()),
  })

  useEffect(() => {
    if (!rulesQuery.data?.data) return
    skipNextSave.current = true
    setRulesConfig({
      rules: rulesQuery.data.data.rules ?? [],
      unmatched_action: rulesQuery.data.data.unmatched_action ?? 'ignore',
    })
  }, [rulesQuery.data])

  const saveMutation = useMutation({
    mutationFn: async (config: UserCleanupRulesConfig) =>
      requireServerSuccess(await saveUserCleanupRules(config)),
    onError: (error) =>
      handleServerError(error, t('Failed to save cleanup rules')),
  })

  const saveRules = saveMutation.mutate
  useEffect(() => {
    if (skipNextSave.current) {
      skipNextSave.current = false
      return
    }
    if (saveTimer.current) window.clearTimeout(saveTimer.current)
    saveTimer.current = window.setTimeout(() => {
      saveRules(rulesConfig)
    }, 500)
    return () => {
      if (saveTimer.current) window.clearTimeout(saveTimer.current)
    }
  }, [rulesConfig, saveRules])

  const groupsQuery = useQuery({
    queryKey: ['groups'],
    queryFn: async () => requireServerSuccess(await getGroups()),
    staleTime: 5 * 60 * 1000,
  })
  const groups = groupsQuery.data?.data ?? []

  const scanQuery = useQuery({
    queryKey: SCAN_QUERY,
    queryFn: async () => requireServerSuccess(await getUserCleanupScan()),
    refetchInterval: (query) =>
      isActiveCleanupScan(query.state.data?.data?.status) ? 800 : false,
  })
  const scanTask = scanQuery.data?.data ?? null
  const scanning = isActiveCleanupScan(scanTask?.status)

  const candidatesQuery = useQuery({
    queryKey: [
      ...CANDIDATES_QUERY,
      candidatePage.pageIndex,
      candidatePage.pageSize,
    ],
    queryFn: async () => {
      const result = await getUserCleanupCandidates({
        p: candidatePage.pageIndex + 1,
        page_size: candidatePage.pageSize,
      })
      if (!result.success) {
        throw createServerError(result, t('Failed to load matched users'))
      }
      return {
        items: result.data?.items ?? [],
        total: result.data?.total ?? 0,
      }
    },
    enabled: !scanning,
  })

  const candidateIdsQuery = useQuery({
    queryKey: CANDIDATE_IDS_QUERY,
    queryFn: async () =>
      requireServerSuccess(await getUserCleanupCandidateIds()),
    enabled: !scanning,
  })
  const allCandidateIds = candidateIdsQuery.data?.data ?? []

  const historyQuery = useQuery({
    queryKey: [...HISTORY_QUERY, historyPage.pageIndex, historyPage.pageSize],
    queryFn: async () => {
      const result = await getUserCleanupHistory({
        p: historyPage.pageIndex + 1,
        page_size: historyPage.pageSize,
      })
      if (!result.success) {
        throw createServerError(result, t('Failed to load cleanup history'))
      }
      return {
        items: result.data?.items ?? [],
        total: result.data?.total ?? 0,
      }
    },
  })

  const scanMutation = useMutation({
    mutationFn: async () => requireServerSuccess(await startUserCleanupScan()),
    onSuccess: async (result) => {
      const skipped = rulesConfig.rules.filter(isIncompleteCleanupRule).length
      if (skipped > 0) {
        toast.warning(t('Incomplete rules were skipped during the scan.'))
      }
      setRowSelection({})
      setCandidatePage((page) => ({ ...page, pageIndex: 0 }))
      queryClient.removeQueries({ queryKey: CANDIDATES_QUERY })
      queryClient.removeQueries({ queryKey: CANDIDATE_IDS_QUERY })
      await queryClient.invalidateQueries({ queryKey: SCAN_QUERY })
      queryClient.setQueryData(SCAN_QUERY, result)
    },
    onError: (error) =>
      handleServerError(error, t('Failed to start user scan')),
  })

  const applyMutation = useMutation({
    mutationFn: async (userIds: number[]) =>
      requireServerSuccess(await applyUserCleanup(userIds)),
    onSuccess: async () => {
      toast.success(t('Users disabled successfully'))
      setRowSelection({})
      setHistoryPage((page) => ({ ...page, pageIndex: 0 }))
      setConfirmOpen(false)
      await queryClient.invalidateQueries({ queryKey: CANDIDATES_QUERY })
      await queryClient.invalidateQueries({ queryKey: CANDIDATE_IDS_QUERY })
      await queryClient.invalidateQueries({ queryKey: HISTORY_QUERY })
      await queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: (error) => handleServerError(error, t('Failed to clean users')),
  })

  const selectedIds = useMemo(
    () =>
      Object.keys(rowSelection)
        .filter((id) => rowSelection[id])
        .map(Number),
    [rowSelection]
  )

  const columns = useUsersColumns({ omitActions: true })
  const { table } = useDataTable({
    data: candidatesQuery.data?.items ?? [],
    columns,
    enableRowSelection: true,
    getRowId: (user) => String(user.id),
    rowSelection,
    onRowSelectionChange: setRowSelection,
    pagination: candidatePage,
    onPaginationChange: (updater) => {
      setCandidatePage((previous) =>
        typeof updater === 'function' ? updater(previous) : updater
      )
    },
    manualPagination: true,
    totalCount: candidatesQuery.data?.total ?? 0,
  })
  const historyColumns = useCleanupHistoryColumns()
  const { table: historyTable } = useDataTable({
    data: historyQuery.data?.items ?? [],
    columns: historyColumns,
    getRowId: (record) => String(record.id),
    pagination: historyPage,
    onPaginationChange: (updater) => {
      setHistoryPage((previous) =>
        typeof updater === 'function' ? updater(previous) : updater
      )
    },
    manualPagination: true,
    totalCount: historyQuery.data?.total ?? 0,
  })

  const progress = Math.min(100, Math.max(0, scanTask?.state?.progress ?? 0))

  const updateRule = (id: string, patch: Partial<UserCleanupRule>) => {
    setRulesConfig((current) => ({
      ...current,
      rules: current.rules.map((rule) =>
        rule.id === id ? { ...rule, ...patch } : rule
      ),
    }))
  }

  const moveRule = (index: number, direction: -1 | 1) => {
    setRulesConfig((current) => ({
      ...current,
      rules: moveCleanupRule(current.rules, index, direction),
    }))
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('User Cleanup')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='flex flex-col gap-6'>
            <section className='space-y-3'>
              <div className='flex items-center justify-between gap-3'>
                <h3 className='text-sm font-medium'>{t('Cleanup rules')}</h3>
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() =>
                    setRulesConfig((current) => ({
                      ...current,
                      rules: [...current.rules, createCleanupRule()],
                    }))
                  }
                >
                  <Plus data-icon='inline-start' />
                  {t('Add rule')}
                </Button>
              </div>
              {rulesConfig.rules.length === 0 && (
                <p className='text-muted-foreground text-sm'>
                  {t(
                    'No cleanup rules yet. Unmatched users use the default action.'
                  )}
                </p>
              )}
              <div className='space-y-2'>
                {rulesConfig.rules.map((rule, index) => (
                  <div
                    key={rule.id}
                    className='flex flex-wrap items-center gap-2 rounded-lg border p-3'
                  >
                    <Select
                      items={[
                        { value: 'group', label: t('User group') },
                        {
                          value: 'idle_time',
                          label: t('Time since last activity'),
                        },
                      ]}
                      value={rule.condition}
                      onValueChange={(value) => {
                        if (value === 'group') {
                          updateRule(rule.id, {
                            condition: 'group',
                            group: rule.group ?? groups[0] ?? '',
                          })
                          return
                        }
                        if (value) {
                          updateRule(rule.id, {
                            condition: value as UserCleanupCondition,
                            operator: rule.operator ?? 'gt',
                            duration: rule.duration ?? 1,
                            unit: rule.unit ?? 'day',
                          })
                        }
                      }}
                    >
                      <SelectTrigger className='w-[200px]'>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='group'>
                            {t('User group')}
                          </SelectItem>
                          <SelectItem value='idle_time'>
                            {t('Time since last activity')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    {rule.condition === 'group' ? (
                      <Combobox
                        options={groups.map((group) => ({
                          value: group,
                          label: group,
                        }))}
                        value={rule.group ?? ''}
                        onValueChange={(value) =>
                          updateRule(rule.id, { group: value ?? '' })
                        }
                        placeholder={t('Select a group')}
                        className='w-[180px]'
                      />
                    ) : (
                      <>
                        <Select
                          items={[
                            { value: 'gt', label: t('Greater than') },
                            { value: 'gte', label: t('Greater than or equal') },
                            { value: 'eq', label: t('Equal') },
                            { value: 'lt', label: t('Less than') },
                            { value: 'lte', label: t('Less than or equal') },
                          ]}
                          value={rule.operator ?? 'gt'}
                          onValueChange={(value) =>
                            value &&
                            updateRule(rule.id, {
                              operator: value as UserCleanupOperator,
                            })
                          }
                        >
                          <SelectTrigger className='w-[180px]'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent alignItemWithTrigger={false}>
                            <SelectGroup>
                              <SelectItem value='gt'>
                                {t('Greater than')}
                              </SelectItem>
                              <SelectItem value='gte'>
                                {t('Greater than or equal')}
                              </SelectItem>
                              <SelectItem value='eq'>{t('Equal')}</SelectItem>
                              <SelectItem value='lt'>
                                {t('Less than')}
                              </SelectItem>
                              <SelectItem value='lte'>
                                {t('Less than or equal')}
                              </SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                        <Input
                          type='number'
                          min={1}
                          className='w-[90px]'
                          value={rule.duration ?? 1}
                          onChange={(event) =>
                            updateRule(rule.id, {
                              duration: Number(event.target.value) || 0,
                            })
                          }
                        />
                        <Select
                          items={[
                            { value: 'hour', label: t('Hours') },
                            { value: 'day', label: t('Days') },
                            { value: 'month', label: t('Months') },
                          ]}
                          value={rule.unit ?? 'day'}
                          onValueChange={(value) =>
                            value &&
                            updateRule(rule.id, {
                              unit: value as UserCleanupUnit,
                            })
                          }
                        >
                          <SelectTrigger className='w-[120px]'>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent alignItemWithTrigger={false}>
                            <SelectGroup>
                              <SelectItem value='hour'>{t('Hours')}</SelectItem>
                              <SelectItem value='day'>{t('Days')}</SelectItem>
                              <SelectItem value='month'>
                                {t('Months')}
                              </SelectItem>
                            </SelectGroup>
                          </SelectContent>
                        </Select>
                      </>
                    )}
                    <Select
                      items={[
                        { value: 'include', label: t('Add to list') },
                        { value: 'ignore', label: t('Ignore user') },
                      ]}
                      value={rule.action}
                      onValueChange={(value) =>
                        value &&
                        updateRule(rule.id, {
                          action: value as UserCleanupAction,
                        })
                      }
                    >
                      <SelectTrigger className='w-[150px]'>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          <SelectItem value='include'>
                            {t('Add to list')}
                          </SelectItem>
                          <SelectItem value='ignore'>
                            {t('Ignore user')}
                          </SelectItem>
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                    <div className='ml-auto flex items-center gap-1'>
                      <Button
                        size='icon-sm'
                        variant='ghost'
                        aria-label={t('Move up')}
                        disabled={index === 0}
                        onClick={() => moveRule(index, -1)}
                      >
                        <ArrowUp />
                      </Button>
                      <Button
                        size='icon-sm'
                        variant='ghost'
                        aria-label={t('Move down')}
                        disabled={index === rulesConfig.rules.length - 1}
                        onClick={() => moveRule(index, 1)}
                      >
                        <ArrowDown />
                      </Button>
                      <Button
                        size='icon-sm'
                        variant='ghost'
                        aria-label={t('Delete rule')}
                        onClick={() =>
                          setRulesConfig((current) => ({
                            ...current,
                            rules: current.rules.filter(
                              (item) => item.id !== rule.id
                            ),
                          }))
                        }
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
              <div className='flex flex-wrap items-center gap-2'>
                <span className='text-sm'>{t('Unmatched users')}</span>
                <Select
                  items={[
                    { value: 'include', label: t('Add to list') },
                    { value: 'ignore', label: t('Ignore user') },
                  ]}
                  value={rulesConfig.unmatched_action}
                  onValueChange={(value) =>
                    value &&
                    setRulesConfig((current) => ({
                      ...current,
                      unmatched_action: value as UserCleanupAction,
                    }))
                  }
                >
                  <SelectTrigger className='w-[180px]'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      <SelectItem value='include'>
                        {t('Add to list')}
                      </SelectItem>
                      <SelectItem value='ignore'>{t('Ignore user')}</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
            </section>

            <section className='space-y-3'>
              {(scanning || scanMutation.isPending) && (
                <div className='space-y-1'>
                  <div className='text-muted-foreground flex items-center justify-between text-xs'>
                    <span>{t('Scanning users...')}</span>
                    <span>{progress}%</span>
                  </div>
                  <Progress value={progress} />
                </div>
              )}
              <Button
                onClick={() => scanMutation.mutate()}
                disabled={scanning || scanMutation.isPending}
              >
                {scanning || scanMutation.isPending
                  ? t('Checking...')
                  : t('Check now')}
              </Button>
              <div className='h-[360px] overflow-hidden rounded-lg border'>
                <DataTablePage
                  table={table}
                  columns={columns}
                  isLoading={candidatesQuery.isLoading}
                  isFetching={candidatesQuery.isFetching}
                  emptyTitle={t('No matched users')}
                  emptyDescription={t('Run a check to fill the cleanup list.')}
                  toolbarProps={null}
                  paginationInFooter={false}
                  fixedHeight
                  applyHeaderSize
                  hideMobile
                />
              </div>
              <div className='flex flex-wrap items-center gap-2'>
                <Button
                  variant='outline'
                  disabled={allCandidateIds.length === 0}
                  onClick={() => {
                    const next: RowSelectionState = {}
                    for (const id of allCandidateIds) next[String(id)] = true
                    setRowSelection(next)
                  }}
                >
                  {t('Select all matched')}
                </Button>
                <Button
                  variant='destructive'
                  disabled={selectedIds.length === 0 || applyMutation.isPending}
                  onClick={() => setConfirmOpen(true)}
                >
                  {t('Clean selected')}
                </Button>
              </div>
            </section>

            <section className='space-y-3'>
              <h3 className='text-sm font-medium'>{t('Cleanup history')}</h3>
              <div className='flex h-[320px] flex-col overflow-hidden rounded-lg border'>
                <DataTablePage
                  table={historyTable}
                  columns={historyColumns}
                  isLoading={historyQuery.isLoading}
                  isFetching={historyQuery.isFetching}
                  emptyTitle={t('No cleanup history')}
                  emptyDescription={t('Cleaned users will appear here.')}
                  toolbarProps={null}
                  paginationInFooter={false}
                  compactPagination
                  fixedHeight
                  applyHeaderSize
                  hideMobile
                />
              </div>
            </section>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>
      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title={t('Clean selected')}
        desc={t('Disable {{count}} selected users?', {
          count: selectedIds.length,
        })}
        confirmText={t('Clean selected')}
        destructive
        isLoading={applyMutation.isPending}
        handleConfirm={() => applyMutation.mutate(selectedIds)}
      />
    </>
  )
}

function useCleanupHistoryColumns(): ColumnDef<UserCleanupRecord>[] {
  const { t } = useTranslation()
  return [
    {
      accessorKey: 'user_id',
      header: t('ID'),
      cell: ({ row }) => (
        <TableId
          value={row.original.user_id}
          className='[font-family:inherit]'
        />
      ),
      size: 80,
    },
    {
      accessorKey: 'username',
      header: t('Username'),
      cell: ({ row }) => (
        <div>
          <div>{row.original.username}</div>
          {row.original.display_name &&
            row.original.display_name !== row.original.username && (
              <div className='text-muted-foreground text-xs'>
                {row.original.display_name}
              </div>
            )}
        </div>
      ),
      size: 180,
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => {
        const status =
          USER_STATUSES[row.original.status as keyof typeof USER_STATUSES]
        if (!status) return '-'
        return (
          <StatusBadge
            label={t(status.labelKey)}
            variant={status.variant}
            copyable={false}
          />
        )
      },
      size: 120,
    },
    {
      accessorKey: 'group',
      header: t('User Group'),
      cell: ({ row }) => <GroupBadge group={row.original.group} />,
      size: 140,
    },
    {
      accessorKey: 'quota',
      header: t('Available Balance'),
      cell: ({ row }) => (
        <span className='tabular-nums'>{formatQuota(row.original.quota)}</span>
      ),
      size: 160,
    },
    {
      accessorKey: 'cleaned_at',
      header: t('Cleanup time'),
      cell: ({ row }) => formatTimestamp(row.original.cleaned_at),
      size: 180,
    },
  ]
}
