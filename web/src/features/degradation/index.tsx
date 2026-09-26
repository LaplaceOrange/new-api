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
import dayjs from 'dayjs'
import { MoreHorizontal, Play, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { EmptyState } from '@/components/empty-state'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ROLE } from '@/lib/roles'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  clearDegradationHistory,
  getDegradationPage,
  getDegradationTest,
  startDegradationTest,
} from './api'
import { DegradationEditor } from './editor'
import type { DegradationModel } from './types'

export function DegradationMonitor() {
  const { t } = useTranslation()
  const role = useAuthStore((state) => state.auth.user?.role ?? 0)
  const isAdmin = role >= ROLE.ADMIN
  const pageQuery = useQuery({
    queryKey: ['degradation'],
    queryFn: async () => requireServerSuccess(await getDegradationPage()),
  })
  const groups = pageQuery.data?.data.groups ?? []
  const config = pageQuery.data?.data.config

  let body = <Skeleton className='h-40 w-full' />
  if (pageQuery.isError) {
    body = (
      <EmptyState
        title={t('Unable to load degradation monitor')}
        description={getServerErrorMessage(pageQuery.error)}
      />
    )
  } else if (pageQuery.data && groups.length === 0) {
    body = (
      <EmptyState
        title={t('No monitored models')}
        description={t('An administrator has not published any groups yet.')}
      />
    )
  } else if (pageQuery.data) {
    const models = groups.flatMap((group) => group.models)
    const counts = models.reduce<Record<string, number>>((acc, model) => {
      acc[model.status] = (acc[model.status] ?? 0) + 1
      return acc
    }, {})
    body = (
      <div className='space-y-6'>
        <div className='grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6'>
          {[
            'passed',
            'retrying',
            'failed',
            'suspected',
            'unavailable',
            'pending',
          ].map((status) => (
            <div
              key={status}
              className='border-border bg-card rounded-xl border p-3 shadow-sm'
            >
              <div className='text-muted-foreground text-xs'>
                {statusLabel(t, status, '')}
              </div>
              <div className='mt-1 text-2xl font-semibold tabular-nums'>
                {counts[status] ?? 0}
              </div>
            </div>
          ))}
        </div>
        {groups.map((group) => (
          <section
            key={group.group}
            className='border-border bg-card overflow-hidden rounded-xl border shadow-sm'
          >
            <div className='flex items-center justify-between border-b px-4 py-3 sm:px-5'>
              <h2 className='text-base font-semibold'>{group.group}</h2>
              <span className='text-muted-foreground text-xs'>
                {group.models.length}
              </span>
            </div>
            {group.models.length === 0 ? (
              <p className='text-muted-foreground px-4 py-6 text-sm sm:px-5'>
                {t('No models')}
              </p>
            ) : (
              <div className='divide-border divide-y'>
                {group.models.map((model) => (
                  <ModelTimeline
                    key={model.model}
                    canClear={isAdmin}
                    group={group.group}
                    model={model}
                  />
                ))}
              </div>
            )}
          </section>
        ))}
      </div>
    )
  }

  return (
    <PublicLayout>
      <PageTransition className='mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-8 sm:px-6'>
        <div>
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('Degradation Monitor')}
          </h1>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Model identity checks by group')}
          </p>
        </div>
        {isAdmin && config ? <DegradationEditor config={config} /> : null}
        {body}
      </PageTransition>
    </PublicLayout>
  )
}

function ModelTimeline(props: {
  canClear: boolean
  group: string
  model: DegradationModel
}) {
  return (
    <div className='grid gap-4 px-4 py-4 sm:px-5 md:grid-cols-[240px_minmax(0,1fr)] md:items-start'>
      <div className='min-w-0 space-y-2'>
        <div className='truncate font-medium'>{props.model.model}</div>
        <StatusMark
          status={props.model.status}
          detected={props.model.detected_model}
        />
      </div>
      <StatusStrip
        canClear={props.canClear}
        events={props.model.timeline}
        group={props.group}
        model={props.model.model}
        unavailable={props.model.status === 'unavailable'}
      />
    </div>
  )
}

const STATUS_STRIP_SLOTS = 20

function StatusStrip(props: {
  canClear: boolean
  events: DegradationModel['timeline']
  group: string
  model: string
  unavailable: boolean
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [taskId, setTaskId] = useState<string | null>(null)
  const events = [...props.events]
    .filter((event) => event.status !== 'retrying')
    .sort((a, b) => a.created_at - b.created_at)
    .slice(-STATUS_STRIP_SLOTS)
  const slots = [
    ...Array.from({ length: STATUS_STRIP_SLOTS - events.length }, () => null),
    ...events,
  ]
  const clearMutation = useMutation({
    mutationFn: async () =>
      requireServerSuccess(
        await clearDegradationHistory(props.group, props.model)
      ),
    onSuccess: async () => {
      setConfirmOpen(false)
      toast.success(t('Detection history cleared'))
      await queryClient.invalidateQueries({ queryKey: ['degradation'] })
    },
    onError: (error) => {
      toast.error(
        getServerErrorMessage(error, t('Failed to clear detection history'))
      )
    },
  })
  const testMutation = useMutation({
    mutationFn: async () =>
      requireServerSuccess(
        await startDegradationTest(props.group, props.model)
      ),
    onSuccess: (response) => {
      setTaskId(response.data.task_id)
      toast.success(t('Test started'))
    },
    onError: (error) => {
      toast.error(getServerErrorMessage(error, t('Test failed')))
    },
  })
  const testQuery = useQuery({
    queryKey: ['degradation-test', taskId],
    queryFn: async () =>
      requireServerSuccess(await getDegradationTest(taskId ?? '')),
    enabled: taskId !== null,
    refetchInterval: (query) =>
      query.state.data?.data.status === 'succeeded' ||
      query.state.data?.data.status === 'failed'
        ? false
        : 1500,
  })
  useEffect(() => {
    if (!taskId) return
    if (testQuery.isError) {
      setTaskId(null)
      return
    }
    if (testQuery.data?.data.status === 'succeeded') {
      toast.success(t('Test completed'))
      void queryClient.invalidateQueries({ queryKey: ['degradation'] })
      setTaskId(null)
    } else if (testQuery.data?.data.status === 'failed') {
      toast.error(testQuery.data.data.error || t('Test failed'))
      void queryClient.invalidateQueries({ queryKey: ['degradation'] })
      setTaskId(null)
    }
  }, [taskId, testQuery.data, testQuery.isError, queryClient, t])
  return (
    <>
      <TooltipProvider>
        <div
          className={`grid h-12 gap-1 ${props.canClear ? 'grid-cols-[repeat(20,minmax(0,1fr))_2.5rem]' : 'grid-cols-20'}`}
        >
          {slots.map((event, index) =>
            event ? (
              <Tooltip
                key={event.created_at + event.status + event.detected_model}
              >
                <TooltipTrigger
                  className={`h-full rounded-sm ${stripColor(event.status)}`}
                  type='button'
                />
                <TooltipContent className='flex flex-col items-start gap-1'>
                  <span className='tabular-nums'>
                    {dayjs.unix(event.created_at).format('YYYY-MM-DD HH:mm')}
                  </span>
                  <span>
                    {statusLabel(t, event.status, event.detected_model)}
                  </span>
                </TooltipContent>
              </Tooltip>
            ) : (
              <span
                key={`empty-${STATUS_STRIP_SLOTS - events.length - index}`}
                className='bg-muted h-full rounded-sm'
              />
            )
          )}
          {props.canClear ? (
            <DropdownMenu modal={false}>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant='ghost'
                    size='icon'
                    className='text-muted-foreground h-12 w-full rounded-sm'
                    aria-label={t('Open menu')}
                  />
                }
              >
                <MoreHorizontal aria-hidden='true' />
              </DropdownMenuTrigger>
              <DropdownMenuContent align='end'>
                <DropdownMenuItem
                  disabled={
                    props.unavailable ||
                    taskId !== null ||
                    testMutation.isPending
                  }
                  onClick={() => testMutation.mutate()}
                >
                  <Play aria-hidden='true' />
                  {t('Test now')}
                </DropdownMenuItem>
                <DropdownMenuItem
                  variant='destructive'
                  disabled={taskId !== null || testMutation.isPending}
                  onClick={() => setConfirmOpen(true)}
                >
                  <Trash2 aria-hidden='true' />
                  {t('Delete all records')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </div>
      </TooltipProvider>
      <ConfirmDialog
        destructive
        handleConfirm={() => clearMutation.mutate()}
        isLoading={clearMutation.isPending}
        onOpenChange={setConfirmOpen}
        open={confirmOpen}
        title={t('Clear detection history')}
        desc={t('Clear all detection results for {{model}}?', {
          model: props.model,
        })}
      />
    </>
  )
}

function stripColor(status: string) {
  if (status === 'passed') return 'bg-emerald-500'
  if (status === 'suspected') return 'bg-amber-400'
  if (status === 'failed') return 'bg-red-500'
  return 'bg-muted-foreground/40'
}

function StatusMark(props: { status: string; detected: string }) {
  const { t } = useTranslation()
  const variants = {
    passed: 'success',
    retrying: 'warning',
    failed: 'danger',
    suspected: 'orange',
    unavailable: 'neutral',
  } as const
  const variant = variants[props.status as keyof typeof variants] ?? 'info'
  return (
    <StatusBadge
      variant={variant}
      copyable={false}
      showDot
      label={statusLabel(t, props.status, props.detected)}
    />
  )
}

function statusLabel(
  t: (key: string, options?: Record<string, unknown>) => string,
  status: string,
  detected: string
) {
  if (status === 'passed') return t('Passed')
  if (status === 'retrying') return t('Retrying')
  if (status === 'failed') return t('Test failed')
  if (status === 'unavailable') return t('Unavailable')
  if (status === 'suspected') {
    return t('Suspected degradation: {{model}}', { model: detected })
  }
  return t('Waiting')
}
