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
import { ArrowRight, BookOpen, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  return (
    <section className='relative z-10 overflow-hidden px-4 py-20 sm:px-6 md:py-28'>
      {/* Background ambient lighting - clean studio atmosphere, zero blue-purple tint */}
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 -z-10 opacity-40 dark:opacity-20'
        style={{
          background:
            'radial-gradient(ellipse 60% 60% at 50% 50%, color-mix(in oklch, var(--foreground) 8%, transparent) 0%, transparent 70%)',
        }}
      />

      <div className='mx-auto max-w-4xl'>
        <AnimateInView
          animation='scale-in'
          className='relative overflow-hidden rounded-3xl border border-border/80 bg-linear-to-b from-card/90 to-card/60 p-8 sm:p-12 md:p-16 text-center shadow-xl backdrop-blur-md dark:border-white/[0.08] dark:from-[#0c1017] dark:to-[#080b11]'
        >
          {/* Subtle top badge */}
          <div className='mb-4 inline-flex items-center gap-1.5 rounded-full border border-border/80 bg-muted/40 px-3 py-1 text-xs font-semibold text-foreground/85'>
            <Sparkles className='size-3 text-amber-500' />
            <span>{t('Instant Access to Top Models')}</span>
          </div>

          <h2 className='text-2xl sm:text-3xl md:text-5xl font-extrabold tracking-tight text-foreground leading-[1.15]'>
            <span>{t('Ready to supercharge')}</span>
            <br />
            <span className='text-muted-foreground'>
              {t('your AI experience?')}
            </span>
          </h2>

          <p className='text-muted-foreground mx-auto mt-4 max-w-xl text-sm sm:text-base leading-relaxed'>
            {t(
              'Sign up now, claim your free trial quota, and experience top-tier AI models in seconds.'
            )}
          </p>

          <div className='mt-8 flex flex-wrap items-center justify-center gap-3.5'>
            {props.isAuthenticated ? (
              <Button
                className='h-11 rounded-xl px-6 text-sm font-semibold shadow-md shadow-primary/15'
                render={<Link to='/dashboard' />}
              >
                <span>{t('Go to Dashboard')}</span>
                <ArrowRight className='ml-2 size-4' />
              </Button>
            ) : (
              <>
                <Button
                  className='group h-11 rounded-xl px-6 text-sm font-semibold shadow-md shadow-primary/15'
                  render={<Link to='/sign-up' />}
                >
                  <span>{t('Get Started Free')}</span>
                  <ArrowRight className='ml-2 size-4 transition-transform duration-200 group-hover:translate-x-1' />
                </Button>
                <Button
                  variant='outline'
                  className='border-border/60 hover:border-border hover:bg-muted/50 h-11 rounded-xl px-5 text-sm font-medium'
                  render={<Link to='/pricing' />}
                >
                  <span>{t('View Pricing')}</span>
                </Button>
              </>
            )}

            {docsUrl.startsWith('http') ? (
              <Button
                variant='ghost'
                className='text-muted-foreground hover:text-foreground h-11 rounded-xl px-4 text-sm font-medium'
                render={
                  <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
                }
              >
                <BookOpen className='mr-1.5 size-4' />
                <span>{t('Docs')}</span>
              </Button>
            ) : (
              <Button
                variant='ghost'
                className='text-muted-foreground hover:text-foreground h-11 rounded-xl px-4 text-sm font-medium'
                render={<Link to={docsUrl} />}
              >
                <BookOpen className='mr-1.5 size-4' />
                <span>{t('Docs')}</span>
              </Button>
            )}
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
