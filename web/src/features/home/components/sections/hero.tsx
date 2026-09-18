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
import { Link } from '@tanstack/react-router'
import {
  Activity,
  ArrowRight,
  BookOpen,
  Code2,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

import { EcosystemMarquee } from '../ecosystem-marquee'
import { HeroTerminalDemo } from '../hero-terminal-demo'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  const renderDocsButton = () => {
    const isExternal = docsUrl.startsWith('http')
    if (isExternal) {
      return (
        <Button
          variant='outline'
          className='border-border/60 hover:border-border hover:bg-muted/50 inline-flex h-11 items-center gap-1.5 rounded-xl px-5 text-sm font-medium transition-colors'
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <BookOpen className='text-muted-foreground size-4' />
          <span>{t('Docs')}</span>
        </Button>
      )
    }
    return (
      <Button
        variant='outline'
        className='border-border/60 hover:border-border hover:bg-muted/50 inline-flex h-11 items-center gap-1.5 rounded-xl px-5 text-sm font-medium transition-colors'
        render={<Link to={docsUrl} />}
      >
        <BookOpen className='text-muted-foreground size-4' />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  return (
    <section className='relative z-10 overflow-hidden px-4 pt-20 pb-8 sm:px-6 md:pt-28 md:pb-16 lg:pt-32 lg:pb-20'>
      {/* Background ambient lighting - clean neutral atmosphere, zero blue-purple tint */}
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10 opacity-60 dark:opacity-40'
        style={{
          background: [
            'radial-gradient(ellipse 70% 45% at 50% -5%, color-mix(in oklch, var(--foreground) 7%, transparent) 0%, transparent 70%)',
          ].join(', '),
        }}
      />

      {/* Subtle grid texture */}
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_70%_50%_at_50%_30%,black_30%,transparent_100%)] bg-[size:3.5rem_3.5rem] opacity-[0.06] dark:opacity-[0.08]'
      />

      <div className='mx-auto max-w-6xl'>
        {/* Main Hero Grid */}
        <div className='grid grid-cols-1 items-center gap-12 lg:grid-cols-12 lg:gap-8'>
          {/* Left Column: Headline, Value Proposition, Action CTAs */}
          <div className='flex flex-col items-start text-left lg:col-span-6'>
            {/* Top Status Badge: sleek terminal-inspired status indicator with live emerald pulse */}
            <div className='landing-animate-fade-up mb-6 inline-flex items-center gap-2.5 rounded-full border border-border/80 bg-muted/40 px-3.5 py-1.5 text-xs font-medium text-muted-foreground backdrop-blur-sm shadow-xs transition-colors hover:border-border hover:bg-muted/60'>
              <span className='relative flex size-2'>
                <span className='absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-500 opacity-75' />
                <span className='relative inline-flex size-2 rounded-full bg-emerald-500' />
              </span>
              <span className='font-mono font-medium text-foreground/90'>
                {t('AI Gateway Engine v1.0')}
              </span>
              <span className='h-3 w-px bg-border/80' />
              <span className='text-muted-foreground'>
                {t('40+ Upstream Providers Active')}
              </span>
            </div>

            {/* Headline: High-impact typography, crisp contrast, zero blue-purple gradient */}
            <h1 className='landing-animate-fade-up text-[clamp(2.2rem,4.2vw,3.6rem)] font-extrabold tracking-tight leading-[1.12]'>
              <span className='text-foreground'>{t('High-Throughput AI Gateway')}</span>
              <br />
              <span className='text-muted-foreground/90 font-bold'>
                {t('One Endpoint. 40+ Providers. Zero Lock-In.')}
              </span>
            </h1>

            {/* Description */}
            <p className='landing-animate-fade-up text-muted-foreground mt-5 max-w-xl text-base leading-relaxed md:text-[15.5px]'>
              {t(
                'Route, balance, and govern requests across OpenAI, Claude, Gemini, DeepSeek, and 40+ providers via a unified OpenAI-compatible endpoint. Built-in dynamic weighted balancing, token-accurate billing, and zero vendor lock-in.'
              )}
            </p>

            {/* Action Buttons */}
            <div className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3.5'>
              {props.isAuthenticated ? (
                <>
                  <Button
                    className='group h-11 rounded-xl px-6 text-sm font-semibold shadow-md shadow-primary/10'
                    render={<Link to='/dashboard' />}
                  >
                    <span>{t('Go to Dashboard')}</span>
                    <ArrowRight className='ml-2 size-4 transition-transform duration-200 group-hover:translate-x-1' />
                  </Button>
                  {renderDocsButton()}
                </>
              ) : (
                <>
                  <Button
                    className='group h-11 rounded-xl px-6 text-sm font-semibold shadow-md shadow-primary/10'
                    render={<Link to='/sign-up' />}
                  >
                    <span>{t('Get Started')}</span>
                    <ArrowRight className='ml-2 size-4 transition-transform duration-200 group-hover:translate-x-1' />
                  </Button>
                  <Button
                    variant='outline'
                    className='border-border/60 hover:border-border hover:bg-muted/50 h-11 rounded-xl px-5 text-sm font-medium transition-colors'
                    render={<Link to='/pricing' />}
                  >
                    <span>{t('View Pricing')}</span>
                  </Button>
                  {renderDocsButton()}
                </>
              )}
            </div>

            {/* Highlights developer micro-chips */}
            <div className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-2'>
              <div className='inline-flex items-center gap-1.5 rounded-lg border border-border/70 bg-card/60 px-2.5 py-1 text-xs font-medium text-foreground/80 font-mono shadow-xs backdrop-blur-xs'>
                <Zap className='size-3.5 text-amber-500' />
                <span>{t('<5ms Overhead')}</span>
              </div>
              <div className='inline-flex items-center gap-1.5 rounded-lg border border-border/70 bg-card/60 px-2.5 py-1 text-xs font-medium text-foreground/80 font-mono shadow-xs backdrop-blur-xs'>
                <ShieldCheck className='size-3.5 text-emerald-500' />
                <span>{t('OWASP ASVS Ready')}</span>
              </div>
              <div className='inline-flex items-center gap-1.5 rounded-lg border border-border/70 bg-card/60 px-2.5 py-1 text-xs font-medium text-foreground/80 font-mono shadow-xs backdrop-blur-xs'>
                <Activity className='size-3.5 text-emerald-500' />
                <span>{t('Auto Health Failover')}</span>
              </div>
              <div className='inline-flex items-center gap-1.5 rounded-lg border border-border/70 bg-card/60 px-2.5 py-1 text-xs font-medium text-foreground/80 font-mono shadow-xs backdrop-blur-xs'>
                <Code2 className='size-3.5 text-foreground/70' />
                <span>{t('OpenAI Drop-In')}</span>
              </div>
            </div>
          </div>

          {/* Right Column: Interactive Hero Terminal Playground */}
          <div className='landing-animate-fade-up w-full lg:col-span-6'>
            <HeroTerminalDemo className='mt-4 lg:mt-0' />
          </div>
        </div>

        {/* Bottom Ecosystem Marquee */}
        <div className='mt-16 md:mt-20 border-t border-border/40 pt-6'>
          <EcosystemMarquee />
        </div>
      </div>
    </section>
  )
}
