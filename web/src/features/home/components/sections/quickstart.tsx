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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Terminal } from 'lucide-react'

import { AnimateInView } from '@/components/animate-in-view'
import { CopyButton } from '@/components/copy-button'
import { cn } from '@/lib/utils'

import { QUICKSTART_SNIPPETS } from '../../constants'

export function Quickstart() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState(0)
  const currentSnippet = QUICKSTART_SNIPPETS[activeTab]

  return (
    <section className='relative z-10 border-t border-border/50 bg-muted/15 px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-5xl'>
        <AnimateInView className='mb-12 text-center'>
          <div className='mb-3 inline-flex items-center gap-1.5 rounded-full border border-border/80 bg-muted/40 px-3 py-1 text-xs font-semibold text-foreground/80'>
            <Terminal className='size-3 text-emerald-500' />
            <span>{t('Developer Quickstart')}</span>
          </div>
          <h2 className='text-2xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-foreground'>
            {t('Integrate in Less Than 60 Seconds')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-3 max-w-xl text-sm sm:text-base leading-relaxed'>
            {t('Works out of the box with your existing OpenAI SDK codebase. Simply update the base URL and authorization key.')}
          </p>
        </AnimateInView>

        <AnimateInView
          delay={100}
          animation='scale-in'
          className='overflow-hidden rounded-2xl border border-border/70 bg-card/90 shadow-lg backdrop-blur-md dark:border-white/[0.08] dark:bg-[#0c1017]'
        >
          {/* Top Bar with Language Tabs and Copy Button */}
          <div className='flex items-center justify-between border-b border-border/60 bg-muted/40 px-4 py-2.5 dark:border-white/[0.06] dark:bg-white/[0.02]'>
            <div className='flex items-center gap-1 sm:gap-2'>
              {QUICKSTART_SNIPPETS.map((snippet, idx) => {
                const isActive = idx === activeTab
                return (
                  <button
                    key={snippet.id}
                    type='button'
                    onClick={() => setActiveTab(idx)}
                    className={cn(
                      'rounded-lg px-3 py-1.5 text-xs font-medium transition-colors',
                      isActive
                        ? 'bg-background text-foreground shadow-xs font-semibold'
                        : 'text-muted-foreground hover:text-foreground'
                    )}
                  >
                    {t(snippet.label)}
                  </button>
                )
              })}
            </div>

            <CopyButton
              value={currentSnippet.code}
              size='sm'
              variant='outline'
              className='h-8 text-xs gap-1.5 rounded-lg border-border/60'
              tooltip={t('Copy code')}
            >
              <span className='hidden sm:inline-block text-[11px] font-medium'>{t('Copy')}</span>
            </CopyButton>
          </div>

          {/* Code Body */}
          <div className='p-4 sm:p-6 overflow-x-auto font-mono text-xs sm:text-[13px] leading-[1.65]'>
            <pre className='text-foreground/90'>
              <code>{currentSnippet.code}</code>
            </pre>
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
