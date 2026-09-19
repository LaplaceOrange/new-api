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
import {
  Sparkles,
  Zap,
  Coins,
  ShieldCheck,
  Code2,
  Palette,
  CheckCircle2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { getLobeIcon } from '@/lib/lobe-icon'

export function Features() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-6xl'>
        {/* Section Header */}
        <AnimateInView className='mb-14 max-w-2xl'>
          <div className='mb-3 inline-flex items-center gap-1.5 rounded-full border border-border/80 bg-muted/40 px-3 py-1 text-xs font-semibold text-foreground/85'>
            <Sparkles className='size-3 text-amber-500' />
            <span>{t('Core Platform Advantages')}</span>
          </div>
          <h2 className='text-2xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-foreground leading-[1.18]'>
            {t('Built for ultimate productivity,')}
            <br />
            <span className='text-muted-foreground'>
              {t('designed for everyone')}
            </span>
          </h2>
          <p className='text-muted-foreground mt-3 text-sm sm:text-base leading-relaxed'>
            {t('Everything you need to accelerate your AI workflow: frontier model access, ultra-low latency, zero subscription waste, and total privacy.')}
          </p>
        </AnimateInView>

        {/* Bento Grid */}
        <div className='grid grid-cols-1 gap-6 md:grid-cols-3'>
          {/* Card 1: All-in-One Frontier Model Access (Span 2) */}
          <AnimateInView
            delay={50}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-2'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-blue-500/10 text-blue-600 dark:text-blue-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Sparkles className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    01
                  </span>
                </div>
                <div className='flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 rounded-full'>
                  <span className='size-1.5 rounded-full bg-emerald-500 animate-ping' />
                  <span>{t('All Models Online')}</span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('All-in-One Frontier Model Access')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Access GPT 6 Astra, DeepSeek V4.1 Flash, Gemini 3.8 Flash, and Grok 4.6 with a single API key. No need to manage multiple international subscriptions or credit cards.')}
              </p>

              {/* Visual Model Matrix */}
              <div className='mt-6 grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs font-mono'>
                {[
                  { name: 'DeepSeek V4.1 Flash', icon: 'DeepSeek.Color', tag: 'Deep Reasoning', status: 'Optimal' },
                  { name: 'GPT 6 Astra', icon: 'OpenAI', tag: 'Next-Gen Omni', status: 'Optimal' },
                  { name: 'Gemini 3.8 Flash', icon: 'Gemini.Color', tag: '1M Context', status: 'Optimal' },
                  { name: 'Grok 4.6', icon: 'Grok.Color', tag: 'Real-Time Insight', status: 'Optimal' },
                ].map((model) => (
                  <div
                    key={model.name}
                    className='flex items-center justify-between rounded-lg border border-border/40 bg-background/70 px-3 py-2 text-foreground/85 transition-colors group-hover:border-border/80'
                  >
                    <div className='flex items-center gap-2'>
                      <div className='flex size-4 items-center justify-center'>
                        {getLobeIcon(model.icon, 16)}
                      </div>
                      <span className='font-medium text-[11.5px]'>{model.name}</span>
                    </div>
                    <span className='text-[10px] text-muted-foreground border border-border/50 bg-muted/40 rounded px-1.5 py-0.5'>
                      {t(model.tag)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </AnimateInView>

          {/* Card 2: Sub-200ms Lightning Response (Span 1) */}
          <AnimateInView
            delay={100}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-1'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-amber-500/10 text-amber-600 dark:text-amber-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Zap className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    02
                  </span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Lightning Fast Response')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-6'>
                {t('Direct high-speed network routing with zero-copy stream chunking for ultra-smooth typing generation.')}
              </p>

              {/* Visual: Latency comparison */}
              <div className='rounded-xl border border-border/50 bg-muted/30 p-4 space-y-3 font-mono text-xs'>
                <div>
                  <div className='flex justify-between text-[11px] mb-1'>
                    <span className='text-muted-foreground'>{t('First Token Response')}</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>&lt; 180 ms</span>
                  </div>
                  <div className='h-2 rounded-full bg-muted/80 overflow-hidden'>
                    <div className='h-full bg-emerald-500 rounded-full w-[25%]' />
                  </div>
                </div>

                <div>
                  <div className='flex justify-between text-[11px] mb-1'>
                    <span className='text-muted-foreground'>{t('Average Streaming Speed')}</span>
                    <span className='text-foreground/80 font-semibold'>90~120 tok/s</span>
                  </div>
                  <div className='h-2 rounded-full bg-muted/80 overflow-hidden'>
                    <div className='h-full bg-blue-500/60 rounded-full w-[85%]' />
                  </div>
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Zero-lag streaming experience')}</span>
            </div>
          </AnimateInView>

          {/* Card 3: Transparent Pay-as-you-go (Span 1) */}
          <AnimateInView
            delay={150}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-1'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-purple-500/10 text-purple-600 dark:text-purple-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Coins className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    03
                  </span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Pay As You Go, No Lock-In')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-5'>
                {t('Pay only for tokens actually consumed. No mandatory monthly subscriptions and top-up balance never expires.')}
              </p>

              {/* Visual: Cost clarity card */}
              <div className='rounded-xl border border-border/50 bg-muted/30 p-3.5 space-y-2 font-mono text-[11px]'>
                <div className='flex justify-between text-[11px] text-foreground font-semibold'>
                  <span>{t('Real-Time Deduction')}</span>
                  <span className='text-emerald-600 dark:text-emerald-400'>{t('100% Transparent')}</span>
                </div>
                <div className='text-muted-foreground text-[10px] leading-relaxed'>
                  {t('Exact token accounting for prompts and completions with live detailed logs in your personal dashboard.')}
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Top-up credits never expire')}</span>
            </div>
          </AnimateInView>

          {/* Card 4: Enterprise Privacy & Zero Retention (Span 2) */}
          <AnimateInView
            delay={200}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-2'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 flex size-9 items-center justify-center rounded-xl border'>
                    <ShieldCheck className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    04
                  </span>
                </div>
                <span className='border border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-[11px] font-semibold px-2.5 py-0.5 rounded-full'>
                  {t('Zero Data Retention')}
                </span>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Enterprise Privacy & Security')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Your conversations and prompts are never stored or used to train upstream models. Full TLS 1.3 encryption and scoped API token security.')}
              </p>

              {/* Visual: Security Highlights */}
              <div className='mt-5 grid grid-cols-1 sm:grid-cols-3 gap-2.5 font-mono text-xs'>
                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>TLS 1.3</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>{t('Encrypted')}</span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>{t('End-to-end transport')}</div>
                </div>

                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>{t('Zero Training')}</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>{t('Guaranteed')}</span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>{t('Prompts never persisted')}</div>
                </div>

                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>{t('Key Isolation')}</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>{t('Active')}</span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>{t('Custom rate & quota limits')}</div>
                </div>
              </div>
            </div>
          </AnimateInView>

          {/* Card 5: 100% Drop-In OpenAI SDK Compatibility (Span 1) */}
          <AnimateInView
            delay={250}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-1'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-sky-500/10 text-sky-600 dark:text-sky-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Code2 className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    05
                  </span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Universal Tool Compatibility')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-4'>
                {t('100% OpenAI-standard compatible. Connect Codex, Cursor, Cherry Studio, Dify, or official SDKs with zero code rewrite.')}
              </p>

              <div className='rounded-xl border border-border/50 bg-muted/30 p-3 font-mono text-[11px] text-foreground/80 space-y-1'>
                <div className='text-muted-foreground/60 text-[10px] uppercase font-sans font-semibold'>
                  {t('1-Line Integration')}
                </div>
                <div className='text-emerald-600 dark:text-emerald-400'>
                  base_url = &quot;https://.../v1&quot;
                </div>
                <div className='text-blue-600 dark:text-blue-400'>
                  api_key = &quot;sk-••••••••••••&quot;
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Compatible with 100+ AI clients')}</span>
            </div>
          </AnimateInView>

          {/* Card 6: Multimodal & Creative Tasks (Span 2) */}
          <AnimateInView
            delay={300}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-2'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-rose-500/10 text-rose-600 dark:text-rose-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Palette className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    06
                  </span>
                </div>
                <span className='border border-rose-500/20 bg-rose-500/10 text-rose-600 dark:text-rose-400 text-[11px] font-semibold px-2.5 py-0.5 rounded-full'>
                  {t('Multimodal Ready')}
                </span>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Multimodal & Creative Generation')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Beyond text chat and code generation: full support for Midjourney imaging, Flux art, audio transcription, and video creation in a single unified account.')}
              </p>

              <div className='mt-4 flex flex-wrap gap-2'>
                {['Midjourney Imagine', 'Flux.1 Schnell', 'DALL-E 3', 'Whisper Audio', 'Suno AI Music'].map((item) => (
                  <span
                    key={item}
                    className='border-border/50 bg-background/70 text-foreground/80 rounded-md border px-2.5 py-1 text-xs font-mono font-medium'
                  >
                    {item}
                  </span>
                ))}
              </div>
            </div>
          </AnimateInView>
        </div>
      </div>
    </section>
  )
}
