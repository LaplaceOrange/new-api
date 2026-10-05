import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Dialog } from '@/components/dialog'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'
import { handleServerError } from '@/lib/handle-server-error'
import { AuthOperationError } from '@/lib/secure-verification'

import { getUpstreamConfig, saveUpstreamConfig } from '../api'
import type { UpstreamConfig } from '../types'

const schema = z.object({
  interval_minutes: z.number().int().min(0).max(1440),
  timezone: z
    .string()
    .trim()
    .refine((value) => {
      try {
        new Intl.DateTimeFormat('en', { timeZone: value })
        return !!value
      } catch {
        return false
      }
    }, 'Enter a valid IANA time zone'),
})

export function UpstreamConfigDialog(props: { onClose: () => void }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['upstreams', 'config'],
    queryFn: ({ signal }) => getUpstreamConfig(signal),
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.onClose()
      }}
      title={t('Refresh settings')}
    >
      {query.isPending && <LoadingState />}
      {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
      {query.data && <ConfigForm config={query.data} onClose={props.onClose} />}
    </Dialog>
  )
}

function ConfigForm(props: { config: UpstreamConfig; onClose: () => void }) {
  const { t } = useTranslation()
  const id = useId()
  const queryClient = useQueryClient()
  const verification = useSecureVerification()
  const [submitting, setSubmitting] = useState(false)
  const form = useForm<UpstreamConfig>({
    resolver: zodResolver(schema),
    defaultValues: props.config,
  })
  const save = useMutation({
    mutationFn: (input: { config: UpstreamConfig; proof: string }) =>
      saveUpstreamConfig(input.config, input.proof),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['upstreams'] })
      props.onClose()
    },
  })
  const submit = async (values: UpstreamConfig) => {
    setSubmitting(true)
    try {
      const proof = await verification.requestVerification({
        scope: 'upstream.credential',
      })
      if (!proof) return
      await save.mutateAsync({ config: values, proof: proof.proof_token })
    } catch (error) {
      handleServerError(AuthOperationError.from(error))
    } finally {
      setSubmitting(false)
    }
  }
  const busy = submitting || save.isPending
  return (
    <>
      <form
        onSubmit={form.handleSubmit((values) => void submit(values))}
        className='space-y-4 text-sm'
      >
        <div className='space-y-2'>
          <Label htmlFor={`${id}-interval`}>
            {t('Refresh interval (minutes)')}
          </Label>
          <Input
            id={`${id}-interval`}
            type='number'
            min={0}
            max={1440}
            disabled={busy}
            aria-invalid={!!form.formState.errors.interval_minutes}
            {...form.register('interval_minutes', { valueAsNumber: true })}
          />
          <p className='text-muted-foreground text-xs'>
            {t('0 disables scheduled refresh')}
          </p>
          {form.formState.errors.interval_minutes && (
            <p role='alert' className='text-destructive text-xs'>
              {t('Enter an interval from 0 to 1440 minutes')}
            </p>
          )}
        </div>
        <div className='space-y-2'>
          <Label htmlFor={`${id}-timezone`}>{t('Time zone')}</Label>
          <Input
            id={`${id}-timezone`}
            disabled={busy}
            aria-invalid={!!form.formState.errors.timezone}
            {...form.register('timezone')}
          />
          {form.formState.errors.timezone && (
            <p role='alert' className='text-destructive text-xs'>
              {t('Enter a valid IANA time zone')}
            </p>
          )}
        </div>
        <div className='flex flex-wrap justify-end gap-2'>
          <Button type='button' variant='outline' onClick={props.onClose}>
            {t('Cancel')}
          </Button>
          <Button type='submit' disabled={busy}>
            {t('Save')}
          </Button>
        </div>
      </form>
      <SecureVerificationDialog {...verification.dialogProps} />
    </>
  )
}
