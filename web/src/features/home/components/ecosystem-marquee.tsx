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
import { Sparkles } from 'lucide-react'
import { memo } from 'react'
import { useTranslation } from 'react-i18next'

import { getLobeIcon } from '@/lib/lobe-icon'

import {
  UPSTREAM_PROVIDERS,
  SUPPORTED_APPS,
  type UpstreamProviderItem,
  type SupportedAppItem,
} from '../constants'

function ProviderCard(props: { item: UpstreamProviderItem }) {
  const { item } = props
  return (
    <div className='group border-border/50 bg-background/80 hover:border-border hover:bg-muted/40 flex shrink-0 items-center gap-3 rounded-full border px-4 py-2 text-xs font-medium shadow-xs backdrop-blur-xs transition-all duration-200 hover:scale-[1.02]'>
      <div className='flex size-6 shrink-0 items-center justify-center'>
        {getLobeIcon(item.icon, 20)}
      </div>
      <span className='text-foreground font-semibold'>{item.name}</span>
      <span className='text-muted-foreground/75 border-border/60 bg-muted/60 hidden rounded-full border px-2 py-0.5 text-[10px] font-normal sm:inline-block'>
        {item.highlightModel}
      </span>
    </div>
  )
}

function AppCard(props: { item: SupportedAppItem }) {
  const { item } = props
  const content = (
    <div className='group border-border/50 bg-background/80 hover:border-border hover:bg-muted/40 flex shrink-0 items-center gap-2.5 rounded-full border px-4 py-2 text-xs font-medium shadow-xs backdrop-blur-xs transition-all duration-200 hover:scale-[1.02]'>
      <div className='flex size-5 shrink-0 items-center justify-center text-blue-500 dark:text-blue-400'>
        {item.icon ? (
          getLobeIcon(item.icon, 18)
        ) : (
          <Sparkles className='size-4' />
        )}
      </div>
      <span className='text-foreground font-medium'>{item.name}</span>
      <span className='text-muted-foreground/75 border-border/50 bg-muted/40 rounded-full border px-1.5 py-0.5 text-[9.5px]'>
        {item.badge}
      </span>
    </div>
  )

  if (item.url) {
    return (
      <a
        href={item.url}
        target='_blank'
        rel='noopener noreferrer'
        className='focus-visible:outline-hidden'
      >
        {content}
      </a>
    )
  }

  return content
}

export const EcosystemMarquee = memo(function EcosystemMarquee() {
  const { t } = useTranslation()

  return (
    <div className='relative w-full overflow-hidden pt-6 pb-2'>
      <div className='mb-3 text-center'>
        <span className='text-muted-foreground/60 text-[11px] font-semibold tracking-wider uppercase'>
          {t('Seamless Multi-Provider & Application Ecosystem')}
        </span>
      </div>

      {/* Edge gradient fade masks */}
      <div className='pointer-events-none absolute inset-y-0 left-0 z-10 w-24 bg-linear-to-r from-background to-transparent' />
      <div className='pointer-events-none absolute inset-y-0 right-0 z-10 w-24 bg-linear-to-l from-background to-transparent' />

      {/* Row 1: Upstream Providers (Scrolls Left) */}
      <div className='marquee-container flex overflow-hidden py-1.5'>
        <div className='animate-marquee-left flex shrink-0 items-center gap-3 pr-3'>
          {UPSTREAM_PROVIDERS.map((provider) => (
            <ProviderCard key={provider.id} item={provider} />
          ))}
        </div>
        {/* Duplicate set for seamless continuous loop */}
        <div
          aria-hidden
          className='animate-marquee-left flex shrink-0 items-center gap-3 pr-3'
        >
          {UPSTREAM_PROVIDERS.map((provider) => (
            <ProviderCard key={`dup-${provider.id}`} item={provider} />
          ))}
        </div>
      </div>

      {/* Row 2: Supported Client Applications (Scrolls Right) */}
      <div className='marquee-container flex overflow-hidden py-1.5'>
        <div className='animate-marquee-right flex shrink-0 items-center gap-3 pr-3'>
          {SUPPORTED_APPS.map((app) => (
            <AppCard key={app.name} item={app} />
          ))}
        </div>
        {/* Duplicate set for seamless continuous loop */}
        <div
          aria-hidden
          className='animate-marquee-right flex shrink-0 items-center gap-3 pr-3'
        >
          {SUPPORTED_APPS.map((app) => (
            <AppCard key={`dup-${app.name}`} item={app} />
          ))}
        </div>
      </div>
    </div>
  )
})
