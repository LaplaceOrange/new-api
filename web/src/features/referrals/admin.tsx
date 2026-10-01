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
import { Pencil, Plus, Save } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { StaticDataTable } from '@/components/data-table'
import { DateTimePicker } from '@/components/datetime-picker'
import { ErrorState } from '@/components/error-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import {
  getReferralAdmin,
  getReferralBuckets,
  saveReferralCampaign,
  saveReferralPolicy,
} from './api'
import { CampaignDialog } from './campaign-dialog'
import { referralPolicySchema } from './form-schema'
import type { ReferralCampaign, ReferralPolicy } from './types'

export function ReferralAdmin() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [editing, setEditing] = useState<ReferralCampaign | null>(null)
  const query = useQuery({
    queryKey: ['referrals', 'admin'],
    queryFn: getReferralAdmin,
  })
  const save = useMutation({
    mutationFn: saveReferralCampaign,
    onSuccess: async () => {
      setEditing(null)
      await client.invalidateQueries({ queryKey: ['referrals'] })
      toast.success(t('Saved successfully'))
    },
  })
  const create = () => {
    const now = Math.floor(Date.now() / 1000)
    setEditing({
      id: 0,
      name: '',
      enabled: false,
      priority: 0,
      start_at: now,
      end_at: now + 30 * 86400,
      direction: 'both',
      spend_basis: 'wallet',
      required_friends: 5,
      paid_threshold_quota: parseQuotaFromDollars(10),
      base_bps: 200,
      inviter_bps: 500,
      invitee_pool_bps: 400,
    })
  }
  const update = async (campaign: ReferralCampaign) => {
    try {
      await save.mutateAsync(campaign)
    } catch (error) {
      handleServerError(error)
    }
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Referral campaigns')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button onClick={create}>
          <Plus aria-hidden='true' />
          {t('Create referral campaign')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        {query.isPending && <LoadingState />}
        {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
        {query.data && (
          <div className='mx-auto max-w-7xl space-y-6'>
            <StaticDataTable
              data={query.data.campaigns ?? []}
              getRowKey={(campaign) => campaign.id}
              emptyContent={t('No referral campaigns')}
              columns={[
                {
                  id: 'name',
                  header: t('Name'),
                  cell: (campaign) => campaign.name,
                },
                {
                  id: 'enabled',
                  header: t('Enabled'),
                  cell: (campaign) => (
                    <Switch
                      checked={campaign.enabled}
                      aria-label={`${t('Enabled')} ${campaign.name}`}
                      disabled={save.isPending}
                      onCheckedChange={(enabled) =>
                        void update({ ...campaign, enabled })
                      }
                    />
                  ),
                },
                {
                  id: 'priority',
                  header: t('Priority'),
                  cell: (campaign) => campaign.priority,
                },
                {
                  id: 'period',
                  header: t('Period'),
                  cell: (campaign) => (
                    <div className='text-xs whitespace-nowrap'>
                      <div>
                        {new Date(campaign.start_at * 1000).toLocaleString()}
                      </div>
                      <div>
                        {new Date(campaign.end_at * 1000).toLocaleString()}
                      </div>
                    </div>
                  ),
                },
                {
                  id: 'rate',
                  header: t('Base rebate (%)'),
                  cell: (campaign) => `${campaign.base_bps / 100}%`,
                },
                {
                  id: 'edit',
                  header: t('Actions'),
                  cell: (campaign) => (
                    <Button
                      size='icon-sm'
                      variant='ghost'
                      aria-label={t('Edit referral campaign')}
                      title={t('Edit referral campaign')}
                      onClick={() => setEditing(campaign)}
                    >
                      <Pencil aria-hidden='true' />
                    </Button>
                  ),
                },
              ]}
            />
            <PolicySettings
              key={`${query.data.policy.min_transfer_quota}:${query.data.policy.bucket_minutes}`}
              policy={query.data.policy}
            />
            <WithdrawalStats
              bucketMinutes={query.data.policy.bucket_minutes}
              total={query.data.total_transferred}
            />
          </div>
        )}
        {editing && (
          <CampaignDialog
            key={editing.id}
            campaign={editing}
            pending={save.isPending}
            onClose={() => setEditing(null)}
            onSave={update}
          />
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function PolicySettings(props: { policy: ReferralPolicy }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [minimum, setMinimum] = useState(
    String(quotaUnitsToDollars(props.policy.min_transfer_quota))
  )
  const [minutes, setMinutes] = useState(String(props.policy.bucket_minutes))
  const policy = {
    min_transfer_quota: parseQuotaFromDollars(Number(minimum)),
    bucket_minutes: Number(minutes),
  }
  const valid =
    minimum !== '' &&
    minutes !== '' &&
    referralPolicySchema.safeParse(policy).success
  const save = useMutation({
    mutationFn: saveReferralPolicy,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['referrals'] })
      toast.success(t('Saved successfully'))
    },
  })
  return (
    <section className='space-y-4 border-t pt-5'>
      <h2 className='text-base font-semibold'>{t('Withdrawal settings')}</h2>
      <FieldGroup className='grid items-end gap-4 sm:grid-cols-3'>
        <Field>
          <FieldLabel htmlFor='referral-minimum'>
            {t('Minimum withdrawal')}
          </FieldLabel>
          <Input
            id='referral-minimum'
            type='number'
            step='any'
            value={minimum}
            onChange={(event) => setMinimum(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor='referral-minutes'>
            {t('Statistics interval (minutes)')}
          </FieldLabel>
          <Input
            id='referral-minutes'
            type='number'
            min={1}
            step={1}
            value={minutes}
            onChange={(event) => setMinutes(event.target.value)}
          />
        </Field>
        <Button
          disabled={!valid || save.isPending}
          onClick={() => save.mutate(policy)}
        >
          <Save aria-hidden='true' />
          {t('Save')}
        </Button>
      </FieldGroup>
      {!valid && (
        <FieldError>
          {t('Enter a whole number within the allowed range')}
        </FieldError>
      )}
    </section>
  )
}

function WithdrawalStats(props: { bucketMinutes: number; total: number }) {
  const { t } = useTranslation()
  const [range] = useState(() => {
    const end = Math.floor(Date.now() / 1000) + 1
    return { start: end - 7 * 86400, end }
  })
  const [start, setStart] = useState(range.start)
  const [end, setEnd] = useState(range.end)
  const valid = start > 0 && end > start && end - start <= 366 * 86400
  const query = useQuery({
    queryKey: ['referrals', 'withdrawals', start, end, props.bucketMinutes],
    queryFn: () => getReferralBuckets(start, end),
    enabled: valid,
  })
  return (
    <section className='space-y-4 border-t pt-5'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <h2 className='text-base font-semibold'>
          {t('Withdrawal statistics')}
        </h2>
        <p>
          {t('Total withdrawn')}:{' '}
          <strong className='tabular-nums'>{formatQuota(props.total)}</strong>
        </p>
      </div>
      <FieldGroup className='grid gap-4 sm:grid-cols-2'>
        {(['start', 'end'] as const).map((name) => (
          <Field key={name}>
            <FieldLabel htmlFor={`referral-report-${name}`}>
              {name === 'start' ? t('Start time') : t('End time')}
            </FieldLabel>
            <DateTimePicker
              id={`referral-report-${name}`}
              ariaLabel={name === 'start' ? t('Start time') : t('End time')}
              value={new Date((name === 'start' ? start : end) * 1000)}
              onChange={(date) => {
                const value = date ? Math.floor(date.getTime() / 1000) : 0
                if (name === 'start') setStart(value)
                else setEnd(value)
              }}
            />
          </Field>
        ))}
      </FieldGroup>
      {!valid && <FieldError>{t('Invalid time range')}</FieldError>}
      {query.isError && <ErrorState onRetry={() => void query.refetch()} />}
      {query.isLoading && <LoadingState />}
      {valid && !query.isLoading && !query.isError && (
        <StaticDataTable
          data={query.data ?? []}
          getRowKey={(bucket) => bucket.bucket}
          emptyContent={t('No withdrawals')}
          columns={[
            {
              id: 'time',
              header: t('Time'),
              cell: (bucket) => new Date(bucket.bucket * 1000).toLocaleString(),
            },
            {
              id: 'amount',
              header: t('Amount'),
              cell: (bucket) => formatQuota(bucket.amount),
            },
            { id: 'count', header: t('Count'), cell: (bucket) => bucket.count },
          ]}
        />
      )}
    </section>
  )
}
