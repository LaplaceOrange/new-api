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
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { TFunction } from 'i18next'
import { Database, Loader2, RefreshCw, Trash2 } from 'lucide-react'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { getLogStorageStats, startLogCleanupByRetention } from '../api'
import type { LogStorageRetention } from '../types'

const RETENTION_OPTIONS = [1, 7, 30, 0]

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1
  )
  return `${new Intl.NumberFormat(undefined, {
    maximumFractionDigits: index === 0 ? 0 : 2,
  }).format(bytes / 1024 ** index)} ${units[index]}`
}

function retentionLabel(days: number, t: TFunction) {
  if (days === 0) return t('Do not retain logs')
  return t('Retain logs for {{days}} days', { days })
}

function getOption(
  options: LogStorageRetention[],
  days: number
): LogStorageRetention {
  return (
    options.find((option) => option.days === days) ?? {
      days,
      cutoff: 0,
      clearable_rows: 0,
      estimated_bytes: 0,
    }
  )
}

export function LogStoragePanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedDays, setSelectedDays] = useState(30)
  const [confirmOpen, setConfirmOpen] = useState(false)

  const storageQuery = useQuery({
    queryKey: ['system-info', 'log-storage'],
    queryFn: async () => {
      const response = await getLogStorageStats()
      if (!response.success || !response.data) {
        throw new Error(response.message || t('We could not load log storage.'))
      }
      return response.data
    },
    staleTime: 30 * 1000,
    retry: false,
  })

  const cleanupMutation = useMutation({
    mutationFn: async () => {
      const response = await startLogCleanupByRetention(selectedDays)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('Failed to clean logs'))
      }
      return response.data
    },
    onSuccess: () => {
      setConfirmOpen(false)
      toast.success(t('Log cleanup task started.'))
      void queryClient.invalidateQueries({
        queryKey: ['system-info', 'log-storage'],
      })
      void queryClient.invalidateQueries({
        queryKey: ['system-info', 'system-tasks'],
      })
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Failed to clean logs')
      )
    },
  })

  const selectedOption = useMemo(
    () => getOption(storageQuery.data?.options ?? [], selectedDays),
    [selectedDays, storageQuery.data?.options]
  )

  let panelContent: ReactNode
  if (storageQuery.isLoading) {
    panelContent = (
      <div className='space-y-3 p-4 sm:p-5'>
        <Skeleton className='h-16 w-full rounded-md' />
        <Skeleton className='h-24 w-full rounded-md' />
      </div>
    )
  } else if (storageQuery.isError) {
    panelContent = (
      <ErrorState
        title={t('We could not load log storage.')}
        description={
          storageQuery.error instanceof Error
            ? storageQuery.error.message
            : undefined
        }
        onRetry={() => void storageQuery.refetch()}
        className='min-h-[220px]'
      />
    )
  } else if (storageQuery.data) {
    panelContent = (
      <div className='space-y-4 p-4 sm:p-5'>
        <div className='grid gap-3 sm:grid-cols-2'>
          <div className='bg-muted/20 rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>
              {t('Total log storage')}
            </div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>
              {formatBytes(storageQuery.data.total_bytes)}
            </div>
            <div className='text-muted-foreground mt-1 text-xs tabular-nums'>
              {t('{{count}} log entries', {
                count: storageQuery.data.total_rows,
              })}
            </div>
          </div>
          <div className='bg-muted/20 rounded-md border p-3'>
            <div className='text-muted-foreground text-xs'>
              {t('Selected cleanup')}
            </div>
            <div className='mt-1 text-xl font-semibold tabular-nums'>
              {formatBytes(selectedOption.estimated_bytes)}
            </div>
            <div className='text-muted-foreground mt-1 text-xs'>
              {t('Estimated space to free')}
              {storageQuery.data.estimated && ` · ${t('Estimated')}`}
            </div>
          </div>
        </div>

        <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
          {RETENTION_OPTIONS.map((days) => {
            const option = getOption(storageQuery.data.options, days)
            const isSelected = selectedDays === days
            return (
              <button
                key={days}
                type='button'
                aria-pressed={isSelected}
                onClick={() => setSelectedDays(days)}
                className={cn(
                  'rounded-md border p-3 text-left transition-colors focus-visible:ring-2 focus-visible:outline-none',
                  isSelected
                    ? 'border-primary bg-primary/5 ring-primary/20 ring-2'
                    : 'hover:bg-muted/50'
                )}
              >
                <div className='flex items-start justify-between gap-2'>
                  <span className='text-sm font-medium'>
                    {retentionLabel(days, t)}
                  </span>
                  {days === 0 && (
                    <Badge variant='outline'>{t('Permanent')}</Badge>
                  )}
                </div>
                <div className='text-muted-foreground mt-2 text-xs'>
                  {t('Estimated clear: {{size}}', {
                    size: formatBytes(option.estimated_bytes),
                  })}
                </div>
                <div className='text-muted-foreground mt-1 text-xs tabular-nums'>
                  {t('{{count}} entries', { count: option.clearable_rows })}
                </div>
              </button>
            )
          })}
        </div>

        <div className='border-destructive/30 bg-destructive/5 flex flex-col gap-3 rounded-md border p-3 sm:flex-row sm:items-center sm:justify-between'>
          <div className='min-w-0'>
            <div className='text-sm font-medium'>
              {retentionLabel(selectedDays, t)}
            </div>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('Logs before this retention window will be permanently deleted.')}
            </p>
            <p className='text-muted-foreground mt-1 text-xs font-medium tabular-nums'>
              {t('Estimated clear: {{size}}', {
                size: formatBytes(selectedOption.estimated_bytes),
              })}
            </p>
          </div>
          <Button
            type='button'
            variant='destructive'
            size='sm'
            onClick={() => setConfirmOpen(true)}
            disabled={
              cleanupMutation.isPending || selectedOption.clearable_rows <= 0
            }
          >
            {cleanupMutation.isPending ? (
              <Loader2
                data-icon='inline-start'
                className='size-3.5 animate-spin'
              />
            ) : (
              <Trash2 data-icon='inline-start' className='size-3.5' />
            )}
            {t('Clear selected logs')}
          </Button>
        </div>
      </div>
    )
  } else {
    panelContent = null
  }

  return (
    <section className='bg-card overflow-hidden rounded-lg border shadow-xs'>
      <div className='flex flex-col gap-3 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5'>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='bg-muted text-muted-foreground inline-flex size-7 shrink-0 items-center justify-center rounded-md'>
            <Database className='size-4' aria-hidden='true' />
          </span>
          <div className='min-w-0'>
            <h3 className='text-sm font-semibold'>{t('Log Storage')}</h3>
            <p className='text-muted-foreground mt-0.5 text-xs'>
              {t('Review log disk usage and permanently remove older entries.')}
            </p>
          </div>
        </div>
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={() => void storageQuery.refetch()}
          disabled={storageQuery.isFetching}
          aria-label={t('Refresh')}
        >
          <RefreshCw
            data-icon='inline-start'
            className={cn(
              'size-3.5',
              storageQuery.isFetching && 'animate-spin'
            )}
            aria-hidden='true'
          />
          {t('Refresh')}
        </Button>
      </div>

      {panelContent}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Confirm log cleanup')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This will permanently delete {{count}} log entries and free about {{size}}.',
                {
                  count: selectedOption.clearable_rows,
                  size: formatBytes(selectedOption.estimated_bytes),
                }
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={cleanupMutation.isPending}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant='destructive'
              onClick={() => cleanupMutation.mutate()}
              disabled={cleanupMutation.isPending}
            >
              {cleanupMutation.isPending ? t('Cleaning...') : t('Delete logs')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  )
}
