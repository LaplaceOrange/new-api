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
  Laptop,
  Cpu,
  Coins,
  Cloud,
  CheckCircle2,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

export function HowItWorks() {
  const { t } = useTranslation()

  const stages = [
    {
      num: '01',
      title: t('Client Applications'),
      subtitle: t('Drop-in Client Support'),
      desc: t('Integrates seamlessly with any desktop app, web UI, IDE plugin, or standard AI SDK.'),
      icon: <Laptop className='size-5 text-blue-500' />,
      badges: ['Cherry Studio', 'Cursor / Cline', 'NextChat', 'OpenAI SDK'],
    },
    {
      num: '02',
      title: t('Smart Gateway Core'),
      subtitle: t('Resilient Routing & Balancing'),
      desc: t('Dynamic weighted load balancing, automatic failover, protocol conversion, and sub-5ms latency.'),
      icon: <Cpu className='size-5 text-indigo-500' />,
      badges: ['Auto Failover', 'Health Check', 'Protocol Translate', 'Rate Limit'],
    },
    {
      num: '03',
      title: t('Billing & Security'),
      subtitle: t('Precision Quotas & ASVS'),
      desc: t('Expression pricing engine, token saturation protection, scoped API tokens, and full audit logs.'),
      icon: <Coins className='size-5 text-purple-500' />,
      badges: ['Dynamic Pricing', 'Quota Guard', 'Key Scoping', 'Audit Trails'],
    },
    {
      num: '04',
      title: t('Upstream Providers'),
      subtitle: t('50+ Global AI Backends'),
      desc: t('Zero-overhead streaming pass-through directly to public clouds, model APIs, or private Ollama nodes.'),
      icon: <Cloud className='size-5 text-emerald-500' />,
      badges: ['OpenAI / Azure', 'Claude / Bedrock', 'Gemini / Vertex', 'DeepSeek / Ollama'],
    },
  ]

  return (
    <section className='relative z-10 border-t border-border/50 px-4 py-20 sm:px-6 md:py-28'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-14 text-center md:mb-18'>
          <div className='mb-3 inline-flex items-center gap-1.5 rounded-full border border-blue-500/20 bg-blue-500/10 px-3 py-1 text-xs font-semibold text-blue-600 dark:text-blue-400'>
            <Zap className='size-3' />
            <span>{t('Architecture Pipeline')}</span>
          </div>
          <h2 className='text-2xl sm:text-3xl md:text-4xl font-extrabold tracking-tight text-foreground'>
            {t('End-to-End Resilient Gateway Architecture')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-3 max-w-2xl text-sm leading-relaxed sm:text-base'>
            {t('From client request to upstream model generation: high-throughput, secure, and transparently auditable.')}
          </p>
        </AnimateInView>

        {/* 4 Pipeline Stages */}
        <div className='grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4 relative'>
          {stages.map((stage, idx) => (
            <AnimateInView
              key={stage.num}
              delay={idx * 90}
              animation='fade-up'
              className='group border-border/60 bg-card/70 hover:border-border hover:bg-card/95 hover:shadow-md relative flex flex-col justify-between rounded-2xl border p-6 backdrop-blur-xs transition-all duration-300'
            >
              <div>
                {/* Header with icon and step number */}
                <div className='flex items-center justify-between mb-4'>
                  <div className='border-border/60 bg-muted/60 flex size-11 items-center justify-center rounded-xl border shadow-2xs'>
                    {stage.icon}
                  </div>
                  <span className='font-mono text-xs font-bold text-muted-foreground/60 border border-border/40 rounded-full px-2 py-0.5'>
                    STAGE {stage.num}
                  </span>
                </div>

                <h3 className='text-base font-bold text-foreground mb-1'>
                  {stage.title}
                </h3>
                <div className='text-xs font-medium text-blue-600 dark:text-blue-400 mb-3'>
                  {stage.subtitle}
                </div>
                <p className='text-muted-foreground text-xs leading-relaxed mb-5'>
                  {stage.desc}
                </p>
              </div>

              {/* Badges */}
              <div className='border-t border-border/40 pt-3 flex flex-wrap gap-1.5'>
                {stage.badges.map((badge) => (
                  <span
                    key={badge}
                    className='border-border/40 bg-muted/40 text-muted-foreground inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10.5px] font-medium'
                  >
                    <CheckCircle2 className='size-2.5 text-emerald-500' />
                    <span>{badge}</span>
                  </span>
                ))}
              </div>
            </AnimateInView>
          ))}
        </div>

        {/* Real-time Gateway Guarantee Banner */}
        <AnimateInView
          delay={350}
          animation='fade-up'
          className='mt-12 rounded-2xl border border-border/60 bg-muted/30 p-5 sm:p-6 backdrop-blur-xs'
        >
          <div className='flex flex-col sm:flex-row items-center justify-between gap-4'>
            <div className='flex items-center gap-3.5'>
              <div className='flex size-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'>
                <ShieldCheck className='size-5' />
              </div>
              <div>
                <div className='text-sm font-bold text-foreground'>
                  {t('Zero Downtime Failover & Circuit Breaker')}
                </div>
                <div className='text-xs text-muted-foreground mt-0.5'>
                  {t('When an upstream provider throttles or experiences outage, requests automatically fail over within milliseconds.')}
                </div>
              </div>
            </div>
            <div className='flex items-center gap-2 text-xs font-semibold text-emerald-600 dark:text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 px-3 py-1.5 rounded-full shrink-0'>
              <span className='size-2 rounded-full bg-emerald-500 animate-pulse' />
              <span>{t('Active Protection')}</span>
            </div>
          </div>
        </AnimateInView>
      </div>
    </section>
  )
}
