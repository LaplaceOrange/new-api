import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  SecureVerificationDialog,
  useSecureVerification,
} from '@/features/auth/secure-verification'
import { handleServerError } from '@/lib/handle-server-error'
import { AuthOperationError } from '@/lib/secure-verification'

import { saveUpstream } from '../api'
import {
  upstreamEditorDefaults,
  upstreamEditorPayload,
  upstreamEditorSchema,
  type UpstreamEditorValues,
} from '../lib/editor'
import type { Upstream } from '../types'

export function UpstreamEditor(props: {
  upstream: Upstream | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const formId = useId()
  const queryClient = useQueryClient()
  const verification = useSecureVerification()
  const cancelVerification = verification.cancel
  const operation = useRef<AbortController | null>(null)
  const [verifying, setVerifying] = useState(false)
  const form = useForm<UpstreamEditorValues>({
    resolver: zodResolver(upstreamEditorSchema),
    defaultValues: upstreamEditorDefaults(props.upstream),
  })
  const addresses = form
    .watch('addresses')
    .split(/\r?\n/)
    .map((item) => item.trim())
    .filter(Boolean)
  const autoRefresh = form.watch('auto_refresh_token')
  const save = useMutation({
    mutationFn: (input: {
      serialized: string
      proof: string
      signal: AbortSignal
    }) =>
      saveUpstream(
        props.upstream?.id ?? null,
        input.serialized,
        input.proof,
        input.signal
      ),
    retry: false,
    meta: { errorToast: false },
  })
  const busy = verifying || save.isPending

  useEffect(
    () => () => {
      operation.current?.abort()
      operation.current = null
      cancelVerification()
    },
    [cancelVerification]
  )

  const close = () => {
    operation.current?.abort()
    operation.current = null
    cancelVerification()
    props.onClose()
  }

  const submit = async (values: UpstreamEditorValues) => {
    if (operation.current) return
    if (!values.access_token.trim() && !props.upstream?.has_access_token) {
      form.setError('access_token', { message: 'Access token is required' })
      return
    }
    if (
      values.auto_refresh_token &&
      !values.refresh_token.trim() &&
      !props.upstream?.has_refresh_token
    ) {
      form.setError('refresh_token', {
        message: 'Refresh token is required for automatic renewal',
      })
      return
    }
    const current = new AbortController()
    operation.current = current
    setVerifying(true)
    try {
      const serialized = JSON.stringify(upstreamEditorPayload(values))
      if (current.signal.aborted) return
      const proof = await verification.requestVerification({
        scope: 'upstream.credential',
      })
      if (!proof || operation.current !== current) return
      await save.mutateAsync({
        serialized,
        proof: proof.proof_token,
        signal: current.signal,
      })
      if (current.signal.aborted) return
      await queryClient.invalidateQueries({ queryKey: ['upstreams'] })
      props.onClose()
    } catch (error) {
      if (!current.signal.aborted) {
        handleServerError(
          AuthOperationError.from(error, 'Failed to save upstream')
        )
      }
    } finally {
      if (operation.current === current) {
        operation.current = null
        setVerifying(false)
      }
    }
  }

  return (
    <>
      <Dialog
        open
        onOpenChange={(open) => {
          if (!open) close()
        }}
        title={props.upstream ? t('Edit upstream') : t('Add upstream')}
        footer={
          <>
            <Button variant='outline' onClick={close}>
              {t('Cancel')}
            </Button>
            <Button type='submit' form={formId} disabled={busy}>
              {busy ? t('Saving...') : t('Save')}
            </Button>
          </>
        }
      >
        <form
          id={formId}
          onSubmit={form.handleSubmit(submit)}
          className='space-y-4 text-sm'
        >
          <div className='space-y-2'>
            <Label htmlFor={`${formId}-name`}>{t('Name')}</Label>
            <Input
              id={`${formId}-name`}
              disabled={busy}
              aria-invalid={!!form.formState.errors.name}
              {...form.register('name')}
            />
            <p className='text-destructive text-xs'>
              {t(form.formState.errors.name?.message ?? '')}
            </p>
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${formId}-addresses`}>
              {t('Upstream addresses (one per line)')}
            </Label>
            <Textarea
              id={`${formId}-addresses`}
              disabled={busy}
              aria-invalid={!!form.formState.errors.addresses}
              {...form.register('addresses')}
            />
            <p className='text-destructive text-xs'>
              {t(form.formState.errors.addresses?.message ?? '')}
            </p>
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${formId}-primary`}>{t('Primary address')}</Label>
            <Select
              value={form.watch('primary_url')}
              onValueChange={(value) =>
                form.setValue('primary_url', value ?? '', {
                  shouldValidate: true,
                })
              }
              disabled={busy}
            >
              <SelectTrigger
                id={`${formId}-primary`}
                className='w-full min-w-0'
                aria-invalid={!!form.formState.errors.primary_url}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {[...new Set(addresses)].map((address) => (
                  <SelectItem key={address} value={address}>
                    <span className='truncate' title={address}>
                      {address}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className='text-destructive text-xs'>
              {t(form.formState.errors.primary_url?.message ?? '')}
            </p>
          </div>
          <div className='flex items-center justify-between gap-3'>
            <Label htmlFor={`${formId}-auto`}>
              {t('Automatically renew JWT')}
            </Label>
            <Switch
              id={`${formId}-auto`}
              checked={autoRefresh}
              onCheckedChange={(value) =>
                form.setValue('auto_refresh_token', value)
              }
              disabled={busy}
            />
          </div>
          <div className='space-y-2'>
            <Label htmlFor={`${formId}-access`}>{t('Access token')}</Label>
            <Input
              id={`${formId}-access`}
              type='password'
              autoComplete='off'
              disabled={busy}
              aria-invalid={!!form.formState.errors.access_token}
              {...form.register('access_token')}
            />
            {props.upstream?.has_access_token && (
              <p className='text-muted-foreground text-xs'>
                {t('Saved token retained when left blank')}
              </p>
            )}
            <p className='text-destructive text-xs'>
              {t(form.formState.errors.access_token?.message ?? '')}
            </p>
          </div>
          {autoRefresh && (
            <div className='space-y-2'>
              <Label htmlFor={`${formId}-refresh`}>{t('Refresh token')}</Label>
              <Input
                id={`${formId}-refresh`}
                type='password'
                autoComplete='off'
                disabled={busy}
                aria-invalid={!!form.formState.errors.refresh_token}
                {...form.register('refresh_token')}
              />
              {props.upstream?.has_refresh_token && (
                <p className='text-muted-foreground text-xs'>
                  {t('Saved token retained when left blank')}
                </p>
              )}
              <p className='text-destructive text-xs'>
                {t(form.formState.errors.refresh_token?.message ?? '')}
              </p>
            </div>
          )}
          <div className='space-y-2'>
            <Label htmlFor={`${formId}-agent`}>
              {t('User-Agent (optional)')}
            </Label>
            <Input
              id={`${formId}-agent`}
              disabled={busy}
              {...form.register('user_agent')}
            />
          </div>
        </form>
      </Dialog>
      <SecureVerificationDialog {...verification.dialogProps} />
    </>
  )
}
