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
import { Share2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import type { ReferralSummary } from '@/features/referrals/types'
import { formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  affiliateLink: string
  onTransfer: () => void
  complianceConfirmed?: boolean
  loading?: boolean
  summary?: ReferralSummary
}

export function AffiliateRewardsCard({
  affiliateLink,
  onTransfer,
  complianceConfirmed = true,
  loading,
  summary,
}: AffiliateRewardsCardProps) {
  const { t } = useTranslation()
  if (loading) {
    return (
      <Card data-card-hover='false' className='bg-muted/20 py-0'>
        <CardContent className='grid gap-4 p-3 sm:p-4 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,0.72fr)_minmax(320px,1.15fr)] lg:items-center'>
          <div>
            <Skeleton className='h-5 w-32' />
            <Skeleton className='mt-2 h-4 w-48' />
          </div>
          <Skeleton className='h-14 rounded-lg' />
          <Skeleton className='h-10 rounded-lg' />
        </CardContent>
      </Card>
    )
  }

  const hasRewards =
    (summary?.balance ?? 0) >= (summary?.min_transfer_quota ?? Infinity)

  return (
    <Card data-card-hover='false' className='bg-muted/20 py-0'>
      <CardContent className='grid gap-3 p-3 sm:gap-4 sm:p-4 lg:grid-cols-[minmax(200px,1fr)_minmax(180px,0.65fr)_minmax(280px,1fr)] lg:items-center'>
        <div className='flex min-w-0 items-center gap-2.5'>
          <IconBadge tone='chart-3'>
            <Share2 />
          </IconBadge>
          <div className='min-w-0'>
            <h3 className='truncate text-sm font-semibold'>
              {t('Referral Program')}
            </h3>
            <p className='text-muted-foreground text-xs break-words'>
              {summary?.campaign?.name ?? t('No active referral campaign')}
            </p>
          </div>
        </div>

        <div className='grid grid-cols-3 gap-1.5 text-center'>
          {[
            [t('Net withdrawable balance'), formatQuota(summary?.balance ?? 0)],
            [t('Total Earned'), formatQuota(summary?.lifetime_earned ?? 0)],
            [
              t('Qualified friends'),
              summary?.campaign
                ? `${summary.qualified_count}/${summary.campaign.required_friends}`
                : '-',
            ],
          ].map(([label, value]) => (
            <div
              key={label}
              className='grid grid-rows-[minmax(3rem,auto)_auto] items-end'
            >
              <div className='text-muted-foreground self-center text-[10px] leading-tight font-medium break-words'>
                {label}
              </div>
              <div className='mt-0.5 truncate text-sm font-semibold tabular-nums'>
                {value}
              </div>
            </div>
          ))}
        </div>

        <div className='flex items-center gap-2'>
          <Input
            value={affiliateLink}
            readOnly
            className='border-muted bg-background/70 h-9 min-w-0 flex-1 font-mono text-xs'
          />
          <CopyButton
            value={affiliateLink}
            variant='outline'
            className='bg-background size-9 shrink-0'
            iconClassName='size-4'
            tooltip={t('Copy referral link')}
            aria-label={t('Copy referral link')}
          />
          {hasRewards && (
            <Button
              onClick={onTransfer}
              disabled={!complianceConfirmed}
              className='h-9 shrink-0 px-3'
              size='sm'
            >
              {t('Transfer to Balance')}
            </Button>
          )}
        </div>
        {!complianceConfirmed ? (
          <p className='text-muted-foreground text-xs lg:col-span-3'>
            {t(
              'Referral reward transfer is disabled until the administrator confirms compliance terms.'
            )}
          </p>
        ) : null}
        {summary?.campaign && (
          <p className='text-muted-foreground text-xs lg:col-span-3'>
            {summary.unlocked ? t('Unlocked') : t('Not unlocked')}
            {' · '}
            {t('Online paid credit threshold')}: {'> '}
            {formatQuota(summary.campaign.paid_threshold_quota)}
            {' · '}
            {t('Invites')}: {summary.invited_count}
            {' · '}
            {t('Legacy rewards')}: {formatQuota(summary.legacy_balance)}
          </p>
        )}
        {summary && !summary.campaign && (
          <p className='text-muted-foreground text-xs lg:col-span-3'>
            {t('Legacy rewards')}: {formatQuota(summary.legacy_balance)}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
