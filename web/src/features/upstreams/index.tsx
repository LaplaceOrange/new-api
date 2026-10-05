import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, RefreshCw, Settings } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import { AuthOperationError } from '@/lib/secure-verification'
import { useAuthStore } from '@/stores/auth-store'

import { deleteUpstream, getUpstreams, refreshUpstreams } from './api'
import { PeriodControls } from './components/period-controls'
import { UpstreamConfigDialog } from './components/upstream-config-dialog'
import { UpstreamEditor } from './components/upstream-editor'
import { UpstreamList } from './components/upstream-list'
import type { Upstream, UpstreamPeriod } from './types'

const emptyItems: Upstream[] = []

export function Upstreams() {
  const { t } = useTranslation()
  const verification = useSecureVerification()
  const root = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.SUPER_ADMIN
  )
  const queryClient = useQueryClient()
  const [period, setPeriod] = useState<UpstreamPeriod>({
    days: 7,
    mode: 'rolling',
  })
  const [editor, setEditor] = useState<{ upstream: Upstream | null } | null>(
    null
  )
  const [deleting, setDeleting] = useState<Upstream | null>(null)
  const [configOpen, setConfigOpen] = useState(false)
  const query = useQuery({
    queryKey: ['upstreams', 'list', period],
    queryFn: ({ signal }) => getUpstreams(period, signal),
    refetchInterval: (current) =>
      current.state.data?.items.some((item) => item.refreshing) ? 15000 : false,
  })
  const refresh = useMutation({
    mutationFn: refreshUpstreams,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['upstreams'] }),
  })
  const remove = useMutation({
    mutationFn: (input: { id: number; proof: string }) =>
      deleteUpstream(input.id, input.proof),
    onSuccess: async () => {
      setDeleting(null)
      await queryClient.invalidateQueries({ queryKey: ['upstreams'] })
    },
  })
  const handleDelete = async () => {
    if (!deleting || remove.isPending) return
    try {
      const proof = await verification.requestVerification({
        scope: 'upstream.credential',
      })
      if (!proof) return
      await remove.mutateAsync({ id: deleting.id, proof: proof.proof_token })
    } catch (error) {
      handleServerError(AuthOperationError.from(error))
    }
  }
  return (
    <SectionPageLayout headingLevel='h1' stackActionsOnMobile>
      <SectionPageLayout.Title>{t('Upstreams')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          variant='outline'
          disabled={
            refresh.isPending ||
            query.data?.items.some((item) => item.refreshing)
          }
          onClick={() => refresh.mutate(undefined)}
        >
          <RefreshCw className='size-4' />
          {t('Refresh all')}
        </Button>
        {root && (
          <>
            <Button variant='outline' onClick={() => setConfigOpen(true)}>
              <Settings className='size-4' />
              {t('Refresh settings')}
            </Button>
            <Button onClick={() => setEditor({ upstream: null })}>
              <Plus className='size-4' />
              {t('Add upstream')}
            </Button>
          </>
        )}
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='space-y-3 tracking-normal'>
          <PeriodControls value={period} onChange={setPeriod} />
          {query.isError ? (
            <ErrorState
              title={t('Failed to load upstreams')}
              onRetry={() => void query.refetch()}
            />
          ) : (
            <UpstreamList
              items={query.data?.items ?? emptyItems}
              loading={query.isPending}
              fetching={query.isFetching}
              root={root}
              refreshPending={refresh.isPending}
              onRefresh={(id) => refresh.mutate(id)}
              onEdit={(upstream) => setEditor({ upstream })}
              onDelete={setDeleting}
            />
          )}
        </div>
        {root && editor && (
          <UpstreamEditor
            key={editor.upstream?.id ?? 'new'}
            upstream={editor.upstream}
            onClose={() => setEditor(null)}
          />
        )}
        {root && configOpen && (
          <UpstreamConfigDialog onClose={() => setConfigOpen(false)} />
        )}
        {root && (
          <ConfirmDialog
            open={!!deleting}
            onOpenChange={(open) => {
              if (!open) {
                setDeleting(null)
                verification.cancel()
              }
            }}
            title={t('Delete upstream')}
            desc={t(
              'Delete upstream {{name}}? Associated channels are not deleted.',
              { name: deleting?.name }
            )}
            destructive
            confirmText={t('Delete')}
            isLoading={remove.isPending}
            handleConfirm={() => void handleDelete()}
          />
        )}
        <SecureVerificationDialog {...verification.dialogProps} />
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
