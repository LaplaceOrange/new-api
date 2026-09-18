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
  Network,
  Zap,
  Calculator,
  ShieldCheck,
  Code2,
  Puzzle,
  CheckCircle2,
  Lock,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

export function Features() {
  const { t } = useTranslation()

  return (
    <section className='relative z-10 px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-6xl'>
        {/* Section Header */}
        <AnimateInView className='mb-14 max-w-2xl'>
          <div className='mb-3 inline-flex items-center gap-1.5 rounded-full border border-blue-500/20 bg-blue-500/10 px-3 py-1 text-xs font-semibold text-blue-600 dark:text-blue-400'>
            <Zap className='size-3' />
            <span>{t('Core Features')}</span>
          </div>
          <h2 className='text-2xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-foreground leading-[1.18]'>
            {t('Built for developers,')}
            <br />
            <span className='landing-gradient-teal-text'>
              {t('designed for scale')}
            </span>
          </h2>
          <p className='text-muted-foreground mt-3 text-sm sm:text-base leading-relaxed'>
            {t('Enterprise-grade routing, billing, and security primitives designed to power mission-critical AI workloads.')}
          </p>
        </AnimateInView>

        {/* Bento Grid */}
        <div className='grid grid-cols-1 gap-6 md:grid-cols-3'>
          {/* Card 1: Multi-Channel Load Balancing (Span 2) */}
          <AnimateInView
            delay={50}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-2'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-blue-500/10 text-blue-600 dark:text-blue-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Network className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    01
                  </span>
                </div>
                <div className='flex items-center gap-1 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 rounded-full'>
                  <span className='size-1.5 rounded-full bg-emerald-500 animate-ping' />
                  <span>{t('Auto-Routing Active')}</span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Intelligent Multi-Channel Load Balancing')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Distribute traffic across multiple API keys, regions, and cloud providers with automatic failover, weighted routing, and dynamic retry.')}
              </p>

              {/* Visual Micro-Component: Channel Health Monitor */}
              <div className='mt-6 space-y-2 rounded-xl border border-border/50 bg-muted/30 p-3 sm:p-4 text-xs font-mono'>
                {[
                  { name: 'OpenAI Direct (US-East)', latency: '42ms', weight: '50%', health: '100%' },
                  { name: 'Azure OpenAI (HK Regional)', latency: '68ms', weight: '30%', health: '100%' },
                  { name: 'Claude Sonnet (AWS Bedrock)', latency: '85ms', weight: '20%', health: '99.8%' },
                ].map((channel) => (
                  <div
                    key={channel.name}
                    className='flex items-center justify-between rounded-lg border border-border/40 bg-background/70 px-3 py-2 text-foreground/85 transition-colors group-hover:border-border/80'
                  >
                    <div className='flex items-center gap-2.5'>
                      <span className='size-2 rounded-full bg-emerald-500' />
                      <span className='font-medium'>{channel.name}</span>
                    </div>
                    <div className='flex items-center gap-3 text-muted-foreground text-[11px] tabular-nums'>
                      <span>{channel.latency}</span>
                      <span className='border-border/60 bg-muted/60 rounded px-1.5 py-0.5 text-[10px] text-foreground/75 font-semibold'>
                        wt {channel.weight}
                      </span>
                      <span className='text-emerald-600 dark:text-emerald-400 font-semibold'>
                        {channel.health}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </AnimateInView>

          {/* Card 2: Sub-Millisecond Engine (Span 1) */}
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
                {t('Sub-Millisecond Engine')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-6'>
                {t('High-throughput Go proxy core with zero-copy stream pass-through.')}
              </p>

              {/* Visual: Latency comparison bar */}
              <div className='rounded-xl border border-border/50 bg-muted/30 p-4 space-y-3 font-mono text-xs'>
                <div>
                  <div className='flex justify-between text-[11px] mb-1'>
                    <span className='text-muted-foreground'>{t('Gateway Overhead')}</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>&lt; 3.2 ms</span>
                  </div>
                  <div className='h-2 rounded-full bg-muted/80 overflow-hidden'>
                    <div className='h-full bg-emerald-500 rounded-full w-[8%]' />
                  </div>
                </div>

                <div>
                  <div className='flex justify-between text-[11px] mb-1'>
                    <span className='text-muted-foreground'>{t('Model Inference Time')}</span>
                    <span className='text-foreground/75 font-semibold'>160 ms</span>
                  </div>
                  <div className='h-2 rounded-full bg-muted/80 overflow-hidden'>
                    <div className='h-full bg-blue-500/60 rounded-full w-[85%]' />
                  </div>
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Zero-copy streaming pipeline')}</span>
            </div>
          </AnimateInView>

          {/* Card 3: Dynamic Billing & Pricing Expressions (Span 1) */}
          <AnimateInView
            delay={150}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-1'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-purple-500/10 text-purple-600 dark:text-purple-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Calculator className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    03
                  </span>
                </div>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Precision Billing Expressions')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-5'>
                {t('Dynamic pricing expressions with tiered thresholds and zero negative-billing saturation guards.')}
              </p>

              {/* Visual: Billing formula */}
              <div className='rounded-xl border border-border/50 bg-muted/30 p-3.5 space-y-2 font-mono text-[11px]'>
                <div className='text-muted-foreground/75 flex justify-between text-[10px] uppercase font-sans font-semibold'>
                  <span>{t('Pricing Formula')}</span>
                  <span className='text-purple-600 dark:text-purple-400'>v1 Engine</span>
                </div>
                <code className='block bg-background/80 border border-border/40 rounded p-2 text-foreground/90 text-[11px] leading-tight break-all'>
                  tokens * model_ratio * group_ratio
                </code>
                <div className='text-muted-foreground text-[10px] flex items-center gap-1.5 pt-1'>
                  <span className='size-1.5 rounded-full bg-purple-500' />
                  <span>{t('Pre-consume & Settle Invariants Guard')}</span>
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Token-accurate settlement')}</span>
            </div>
          </AnimateInView>

          {/* Card 4: OWASP ASVS Enterprise Security (Span 2) */}
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
                  OWASP ASVS Compliant
                </span>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Enterprise Security & Granular Access')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Scoped API keys, IP allowlists, TOTP/WebAuthn Passkeys, session rotation, and non-secret audit logs.')}
              </p>

              {/* Visual: Scoped Permission Matrix */}
              <div className='mt-5 grid grid-cols-1 sm:grid-cols-3 gap-2.5 font-mono text-xs'>
                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>chat:write</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>✓ Allowed</span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>Standard completions</div>
                </div>

                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>models:read</span>
                    <span className='text-emerald-600 dark:text-emerald-400 font-bold'>✓ Allowed</span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>List accessible models</div>
                </div>

                <div className='rounded-lg border border-border/50 bg-background/70 p-2.5'>
                  <div className='flex items-center justify-between text-[11px] mb-1'>
                    <span className='text-foreground font-semibold'>admin:settings</span>
                    <span className='text-rose-500 font-bold flex items-center gap-0.5'>
                      <Lock className='size-2.5' /> Denied
                    </span>
                  </div>
                  <div className='text-[10px] text-muted-foreground'>Strictly isolated</div>
                </div>
              </div>
            </div>
          </AnimateInView>

          {/* Card 5: 100% SDK Compatibility (Span 1) */}
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
                {t('100% Drop-In SDK Support')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed mb-4'>
                {t('Zero code changes. Simply modify baseURL to unlock multi-model switching.')}
              </p>

              <div className='rounded-xl border border-border/50 bg-muted/30 p-3 font-mono text-[11px] text-foreground/80 space-y-1'>
                <div className='text-muted-foreground/60 text-[10px] uppercase font-sans font-semibold'>
                  {t('1-Line Integration')}
                </div>
                <div className='text-emerald-600 dark:text-emerald-400'>
                  client.base_url = &quot;.../v1&quot;
                </div>
                <div className='text-blue-600 dark:text-blue-400'>
                  client.api_key = &quot;sk-newapi...&quot;
                </div>
              </div>
            </div>

            <div className='mt-4 flex items-center gap-1.5 text-xs text-muted-foreground'>
              <CheckCircle2 className='size-3.5 text-emerald-500' />
              <span>{t('Compatible with all OpenAI SDKs')}</span>
            </div>
          </AnimateInView>

          {/* Card 6: Task Plugins & Extensions (Span 2 on mobile, or 2 on desktop) */}
          <AnimateInView
            delay={300}
            animation='fade-up'
            className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 sm:p-7 backdrop-blur-xs transition-all duration-300 md:col-span-2'
          >
            <div>
              <div className='flex items-center justify-between mb-4'>
                <div className='flex items-center gap-2.5'>
                  <div className='border-border/60 bg-rose-500/10 text-rose-600 dark:text-rose-400 flex size-9 items-center justify-center rounded-xl border'>
                    <Puzzle className='size-5' />
                  </div>
                  <span className='font-mono text-xs font-semibold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    06
                  </span>
                </div>
                <span className='border border-rose-500/20 bg-rose-500/10 text-rose-600 dark:text-rose-400 text-[11px] font-semibold px-2.5 py-0.5 rounded-full'>
                  Sobek JS Sandbox
                </span>
              </div>

              <h3 className='text-base sm:text-lg font-bold text-foreground mb-1.5'>
                {t('Extensible Task Plugin Engine')}
              </h3>
              <p className='text-muted-foreground text-xs sm:text-sm leading-relaxed max-w-xl'>
                {t('Execute media generation and complex asynchronous jobs (Midjourney, Suno Music, Kling Video) via sandboxed JavaScript plugins with automatic polling and settlement.')}
              </p>

              <div className='mt-4 flex flex-wrap gap-2'>
                {['Midjourney Plus', 'Suno AI Music', 'Kling Video', 'Async Polling', 'Unit Price Settlement'].map((item) => (
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
