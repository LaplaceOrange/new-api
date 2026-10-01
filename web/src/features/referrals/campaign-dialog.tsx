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
import { zodResolver } from '@hookform/resolvers/zod'
import { Save } from 'lucide-react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Switch } from '@/components/ui/switch'
import { parseQuotaFromDollars, quotaUnitsToDollars } from '@/lib/format'

import { referralCampaignSchema } from './form-schema'
import type { ReferralCampaign } from './types'

export function CampaignDialog(props: {
  campaign: ReferralCampaign
  pending: boolean
  onClose: () => void
  onSave: (campaign: ReferralCampaign) => Promise<void>
}) {
  const { t } = useTranslation()
  const form = useForm<ReferralCampaign>({
    resolver: zodResolver(referralCampaignSchema),
    defaultValues: props.campaign,
  })

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !props.pending) props.onClose()
      }}
      title={
        props.campaign.id
          ? t('Edit referral campaign')
          : t('Create referral campaign')
      }
      footer={
        <Button
          type='submit'
          form='referral-campaign-form'
          disabled={props.pending}
        >
          <Save aria-hidden='true' />
          {t('Save')}
        </Button>
      }
    >
      <form
        id='referral-campaign-form'
        onSubmit={form.handleSubmit(props.onSave)}
      >
        <FieldGroup className='grid gap-4 sm:grid-cols-2'>
          <Field className='sm:col-span-2'>
            <FieldLabel htmlFor='referral-name'>{t('Name')}</FieldLabel>
            <Input
              id='referral-name'
              {...form.register('name')}
              aria-invalid={!!form.formState.errors.name}
            />
            <FieldError>
              {form.formState.errors.name && t('Invalid campaign fields')}
            </FieldError>
          </Field>
          <Field>
            <FieldLabel htmlFor='referral-priority'>{t('Priority')}</FieldLabel>
            <Input
              id='referral-priority'
              type='number'
              step={1}
              {...form.register('priority', { valueAsNumber: true })}
            />
          </Field>
          <Field orientation='horizontal'>
            <FieldLabel htmlFor='referral-enabled'>{t('Enabled')}</FieldLabel>
            <Controller
              control={form.control}
              name='enabled'
              render={({ field }) => (
                <Switch
                  id='referral-enabled'
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  disabled={!props.campaign.id}
                />
              )}
            />
          </Field>
          {(['start_at', 'end_at'] as const).map((name) => (
            <Field key={name}>
              <FieldLabel htmlFor={`referral-${name}`}>
                {name === 'start_at' ? t('Start time') : t('End time')}
              </FieldLabel>
              <Controller
                control={form.control}
                name={name}
                render={({ field }) => (
                  <DateTimePicker
                    id={`referral-${name}`}
                    ariaLabel={
                      name === 'start_at' ? t('Start time') : t('End time')
                    }
                    value={
                      field.value ? new Date(field.value * 1000) : undefined
                    }
                    onChange={(date) =>
                      field.onChange(
                        date ? Math.floor(date.getTime() / 1000) : 0
                      )
                    }
                  />
                )}
              />
              <FieldError>
                {form.formState.errors[name] &&
                  t('End time must be after start time')}
              </FieldError>
            </Field>
          ))}
          <Field>
            <FieldLabel htmlFor='referral-direction'>
              {t('Rebate direction')}
            </FieldLabel>
            <NativeSelect
              id='referral-direction'
              {...form.register('direction')}
            >
              <NativeSelectOption value='invitee_to_inviter'>
                {t('To inviter')}
              </NativeSelectOption>
              <NativeSelectOption value='inviter_to_invitee'>
                {t('To invited friends')}
              </NativeSelectOption>
              <NativeSelectOption value='both'>
                {t('Both directions')}
              </NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor='referral-basis'>
              {t('Consumption basis')}
            </FieldLabel>
            <NativeSelect id='referral-basis' {...form.register('spend_basis')}>
              <NativeSelectOption value='wallet'>
                {t('Net wallet consumption')}
              </NativeSelectOption>
              <NativeSelectOption value='all'>
                {t('Net wallet and subscription consumption')}
              </NativeSelectOption>
              <NativeSelectOption value='paid_wallet'>
                {t('Tracked paid wallet consumption')}
              </NativeSelectOption>
            </NativeSelect>
          </Field>
          <Field>
            <FieldLabel htmlFor='referral-friends'>
              {t('Required qualified friends')}
            </FieldLabel>
            <Input
              id='referral-friends'
              type='number'
              min={1}
              step={1}
              {...form.register('required_friends', { valueAsNumber: true })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor='referral-threshold'>
              {t('Online paid credit threshold')}
            </FieldLabel>
            <Controller
              control={form.control}
              name='paid_threshold_quota'
              render={({ field }) => (
                <Input
                  id='referral-threshold'
                  type='number'
                  min={0}
                  step='any'
                  value={quotaUnitsToDollars(field.value)}
                  onChange={(event) =>
                    field.onChange(
                      parseQuotaFromDollars(event.target.valueAsNumber)
                    )
                  }
                />
              )}
            />
          </Field>
          {(['base_bps', 'inviter_bps', 'invitee_pool_bps'] as const).map(
            (name) => (
              <Field key={name}>
                <FieldLabel htmlFor={`referral-${name}`}>
                  {name === 'base_bps' && t('Base rebate (%)')}
                  {name === 'inviter_bps' && t('Unlocked inviter rebate (%)')}
                  {name === 'invitee_pool_bps' &&
                    t('Unlocked invited friends pool (%)')}
                </FieldLabel>
                <Controller
                  control={form.control}
                  name={name}
                  render={({ field }) => (
                    <Input
                      id={`referral-${name}`}
                      type='number'
                      min={0}
                      max={100}
                      step={0.01}
                      value={
                        Number.isFinite(field.value) ? field.value / 100 : ''
                      }
                      onChange={(event) =>
                        field.onChange(
                          Math.round(event.target.valueAsNumber * 100)
                        )
                      }
                    />
                  )}
                />
                <FieldError>
                  {form.formState.errors[name] && t('Invalid campaign fields')}
                </FieldError>
              </Field>
            )
          )}
        </FieldGroup>
        {Object.keys(form.formState.errors).length > 0 && (
          <FieldError>{t('Invalid campaign fields')}</FieldError>
        )}
      </form>
    </Dialog>
  )
}
