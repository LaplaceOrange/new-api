import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { PasswordInput } from '@/components/password-input'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import type { UpstreamEditorValues } from '../lib/editor'
import type { Upstream } from '../types'

export function UpstreamCredentialsFields(props: {
  form: UseFormReturn<UpstreamEditorValues>
  formId: string
  busy: boolean
  upstream: Upstream | null
}) {
  const { t } = useTranslation()
  const mode = props.form.watch('auth_mode')
  const autoRefresh = props.form.watch('auto_refresh_token')
  const errors = props.form.formState.errors
  const hasAccount =
    props.upstream?.auth_mode === 'password' &&
    props.upstream.has_account_credentials
  const hasJWT =
    (props.upstream?.auth_mode ?? 'jwt') === 'jwt' &&
    props.upstream?.has_access_token

  return (
    <FieldGroup className='gap-4'>
      <Field>
        <FieldTitle id={`${props.formId}-auth-mode`}>
          {t('Authentication method')}
        </FieldTitle>
        <ToggleGroup
          value={[mode]}
          variant='outline'
          disabled={props.busy}
          aria-labelledby={`${props.formId}-auth-mode`}
          onValueChange={(values) => {
            const next = values[0]
            if (next !== 'jwt' && next !== 'password') return
            props.form.setValue('auth_mode', next)
            props.form.setValue('access_token', '')
            props.form.setValue('refresh_token', '')
            props.form.setValue('account_email', '')
            props.form.setValue('account_password', '')
            props.form.clearErrors([
              'access_token',
              'refresh_token',
              'account_email',
              'account_password',
            ])
          }}
        >
          <ToggleGroupItem value='password'>
            {t('Account password')}
          </ToggleGroupItem>
          <ToggleGroupItem value='jwt'>JWT</ToggleGroupItem>
        </ToggleGroup>
      </Field>
      {mode === 'password' ? (
        <>
          <Field
            data-invalid={!!errors.account_email}
            data-disabled={props.busy}
          >
            <FieldLabel htmlFor={`${props.formId}-email`}>
              {t('Account email')}
            </FieldLabel>
            <Input
              id={`${props.formId}-email`}
              type='email'
              autoComplete='off'
              disabled={props.busy}
              aria-invalid={!!errors.account_email}
              {...props.form.register('account_email')}
            />
            {hasAccount && (
              <FieldDescription>
                {t('Saved account retained when left blank')}
              </FieldDescription>
            )}
            <FieldError>{t(errors.account_email?.message ?? '')}</FieldError>
          </Field>
          <Field
            data-invalid={!!errors.account_password}
            data-disabled={props.busy}
          >
            <FieldLabel htmlFor={`${props.formId}-password`}>
              {t('Account password')}
            </FieldLabel>
            <PasswordInput
              id={`${props.formId}-password`}
              autoComplete='new-password'
              disabled={props.busy}
              aria-invalid={!!errors.account_password}
              {...props.form.register('account_password')}
            />
            {hasAccount && (
              <FieldDescription>
                {t('Saved password retained when left blank')}
              </FieldDescription>
            )}
            <FieldError>{t(errors.account_password?.message ?? '')}</FieldError>
          </Field>
        </>
      ) : (
        <>
          <Field orientation='horizontal' data-disabled={props.busy}>
            <FieldLabel htmlFor={`${props.formId}-auto`}>
              {t('Automatically renew JWT')}
            </FieldLabel>
            <Switch
              id={`${props.formId}-auto`}
              checked={autoRefresh}
              onCheckedChange={(value) =>
                props.form.setValue('auto_refresh_token', value)
              }
              disabled={props.busy}
            />
          </Field>
          <Field
            data-invalid={!!errors.access_token}
            data-disabled={props.busy}
          >
            <FieldLabel htmlFor={`${props.formId}-access`}>
              {t('Access token')}
            </FieldLabel>
            <PasswordInput
              id={`${props.formId}-access`}
              autoComplete='off'
              disabled={props.busy}
              aria-invalid={!!errors.access_token}
              {...props.form.register('access_token')}
            />
            {hasJWT && (
              <FieldDescription>
                {t('Saved token retained when left blank')}
              </FieldDescription>
            )}
            <FieldError>{t(errors.access_token?.message ?? '')}</FieldError>
          </Field>
          {autoRefresh && (
            <Field
              data-invalid={!!errors.refresh_token}
              data-disabled={props.busy}
            >
              <FieldLabel htmlFor={`${props.formId}-refresh`}>
                {t('Refresh token')}
              </FieldLabel>
              <PasswordInput
                id={`${props.formId}-refresh`}
                autoComplete='off'
                disabled={props.busy}
                aria-invalid={!!errors.refresh_token}
                {...props.form.register('refresh_token')}
              />
              {hasJWT && props.upstream?.has_refresh_token && (
                <FieldDescription>
                  {t('Saved token retained when left blank')}
                </FieldDescription>
              )}
              <FieldError>{t(errors.refresh_token?.message ?? '')}</FieldError>
            </Field>
          )}
        </>
      )}
    </FieldGroup>
  )
}
