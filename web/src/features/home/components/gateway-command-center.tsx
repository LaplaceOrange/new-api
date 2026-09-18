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
  ArrowRight,
  Calculator,
  CheckCircle2,
  ChevronRight,
  ExternalLink,
  Laptop,
  Sparkles,
  Zap,
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

type ExperienceTab = 'models' | 'apps' | 'calculator'

interface GatewayCommandCenterProps {
  className?: string
}

interface ShowcaseModel {
  id: string
  name: string
  provider: string
  icon: string
  tagKey: string
  speed: string
  latency: string
  context: string
  savings: string
  prompts: {
    titleKey: string
    promptKey: string
    responseCode?: string
    responseText?: string
  }[]
}

const SHOWCASE_MODELS: ShowcaseModel[] = [
  {
    id: 'deepseek-r1',
    name: 'DeepSeek R1',
    provider: 'DeepSeek',
    icon: 'DeepSeek.Color',
    tagKey: 'Deep Reasoning & Extreme Value',
    speed: '115 tok/s',
    latency: '140ms',
    context: '64K',
    savings: '92% cheaper than official o1',
    prompts: [
      {
        titleKey: 'Concurrency Worker Pool',
        promptKey: 'Write a high-performance Go worker pool with context cancellation and graceful shutdown.',
        responseCode: `// High-concurrency worker pool with context propagation
type WorkerPool struct {
    tasks   chan func()
    workers int
    wg      sync.WaitGroup
}

func (wp *WorkerPool) Run(ctx context.Context) {
    for i := range wp.workers {
        wp.wg.Add(1)
        go func(id int) {
            defer wp.wg.Done()
            for {
                select {
                case task, ok := <-wp.tasks:
                    if !ok { return }
                    task()
                case <-ctx.Done():
                    return
                }
            }
        }(i)
    }
}`,
      },
      {
        titleKey: 'Mathematical Reasoning',
        promptKey: 'Evaluate the optimal routing strategy for distributed API clusters with uneven packet loss.',
        responseText: `<think>
1. Model the network as a weighted graph with dynamic packet loss probabilities.
2. Formulate dynamic channel weighting: W_i = (1 - loss_i) * (1 / (latency_i + epsilon)).
3. Apply exponential moving average (EMA) to prevent route flapping during transient drops.
</think>
The optimal strategy combines latency-based weighted fair queuing (WFQ) with instant circuit-breaker fallback when health checks fail 3 consecutive probes.`,
      },
    ],
  },
  {
    id: 'claude-3-7-sonnet',
    name: 'Claude 3.7 Sonnet',
    provider: 'Anthropic',
    icon: 'Claude.Color',
    tagKey: 'Top-Tier Coding & Hybrid Reasoning',
    speed: '88 tok/s',
    latency: '175ms',
    context: '200K',
    savings: 'Pay-as-you-go, no $20/mo lock-in',
    prompts: [
      {
        titleKey: 'Architecture Refactoring',
        promptKey: 'Explain how to decouple monolith state machines into pure event-driven sagas.',
        responseText: `To transition from an orchestrator monolith to an event-driven saga:

1. **Define Compensating Transactions**: Every mutating step (e.g. ReserveStock, BillQuota) must have an idempotent rollback (ReleaseStock, RefundQuota).
2. **Outbox Pattern**: Write domain events to an outbox table in the same DB transaction as local changes to guarantee at-least-once publishing.
3. **Partitioned Event Log**: Route events by entity ID (e.g., userId) so sequential consistency is preserved without global locks.`,
      },
      {
        titleKey: 'Type-Safe API Router',
        promptKey: 'Design a type-safe RPC client wrapper with compile-time route validation.',
        responseCode: `// Compile-time verified RPC interface
type Endpoint<TReq, TRes> = {
  path: string;
  method: 'GET' | 'POST';
  _req?: TReq;
  _res?: TRes;
};

export async function request<TReq, TRes>(
  ep: Endpoint<TReq, TRes>,
  data: TReq
): Promise<TRes> {
  const res = await fetch(ep.path, {
    method: ep.method,
    body: JSON.stringify(data),
    headers: { 'Content-Type': 'application/json' },
  });
  return res.json() as Promise<TRes>;
}`,
      },
    ],
  },
  {
    id: 'gpt-4o',
    name: 'GPT-4o',
    provider: 'OpenAI',
    icon: 'OpenAI',
    tagKey: 'Omni Multimodal & Fast Vision',
    speed: '95 tok/s',
    latency: '155ms',
    context: '128K',
    savings: 'Transparent pay-per-token',
    prompts: [
      {
        titleKey: 'Fullstack Solution',
        promptKey: 'Generate a production-ready streaming SSE handler in Next.js App Router.',
        responseCode: `// app/api/chat/route.ts (Next.js Edge / Node.js)
import { OpenAI } from 'openai';

const client = new OpenAI({
  baseURL: 'https://api.yourdomain.com/v1',
  apiKey: process.env.API_KEY,
});

export async function POST(req: Request) {
  const { messages } = await req.json();
  const stream = await client.chat.completions.create({
    model: 'gpt-4o',
    messages,
    stream: true,
  });

  return new Response(stream.toReadableStream(), {
    headers: { 'Content-Type': 'text/event-stream' },
  });
}`,
      },
    ],
  },
  {
    id: 'gemini-2-5-pro',
    name: 'Gemini 2.5 Pro',
    provider: 'Google',
    icon: 'Gemini.Color',
    tagKey: '1M+ Massive Context & Reasoning',
    speed: '90 tok/s',
    latency: '160ms',
    context: '1M+',
    savings: 'Massive context with zero loss',
    prompts: [
      {
        titleKey: 'Repo-Wide Code Review',
        promptKey: 'Scan 50,000 lines of codebase for concurrency race conditions and memory leaks.',
        responseText: `Gemini 2.5 Pro processes your entire repository context in a single prompt:

• Identified 2 data races in background goroutine closures (retaining outer loop variables).
• Located 1 unclosed Response.Body in upstream relay proxy client.
• Verified all SQL transaction rollback paths properly invoke tx.Rollback() on error.`,
      },
    ],
  },
]

export function GatewayCommandCenter({ className }: GatewayCommandCenterProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<ExperienceTab>('models')
  const [selectedModelIndex, setSelectedModelIndex] = useState(0)
  const [selectedPromptIndex, setSelectedPromptIndex] = useState(0)
  const [selectedApp, setSelectedApp] = useState<'cursor' | 'cherry' | 'nextchat' | 'python'>('cursor')
  const [calcDailyMessages, setCalcDailyMessages] = useState(50)
  const [calcModelTier, setCalcModelTier] = useState<'deepseek' | 'claude' | 'gpt4o'>('claude')

  const baseUrl = useMemo(() => {
    if (typeof window !== 'undefined' && window.location.origin) {
      return `${window.location.origin}/v1`
    }
    return 'https://api.yourdomain.com/v1'
  }, [])

  const currentModel = SHOWCASE_MODELS[selectedModelIndex]
  const currentPrompt = currentModel.prompts[selectedPromptIndex] || currentModel.prompts[0]

  // Calculator cost estimation
  const calculatedCost = useMemo(() => {
    // Average tokens per message ~ 800 tokens (300 in, 500 out)
    const monthlyCalls = calcDailyMessages * 30
    let perThousandCost = 0.0008 // default deepseek
    if (calcModelTier === 'claude') perThousandCost = 0.004
    if (calcModelTier === 'gpt4o') perThousandCost = 0.003

    const estimatedSpend = (monthlyCalls * perThousandCost).toFixed(2)
    const officialSpend = (20.0).toFixed(2)
    const savingsPercent = Math.max(
      10,
      Math.round(((20.0 - Number.parseFloat(estimatedSpend)) / 20.0) * 100)
    )

    return {
      spend: estimatedSpend,
      official: officialSpend,
      savings: savingsPercent,
    }
  }, [calcDailyMessages, calcModelTier])

  return (
    <div
      className={cn(
        'relative mx-auto w-full max-w-5xl rounded-2xl border border-border/80 bg-card/80 shadow-2xl backdrop-blur-xl transition-all duration-300',
        'dark:border-white/[0.09] dark:bg-[#0b0f17]/90 dark:shadow-[0_20px_60px_-15px_rgba(0,0,0,0.8)]',
        className
      )}
    >
      {/* Interactive Hub Window Header */}
      <div className='flex flex-wrap items-center justify-between border-b border-border/60 px-4 py-3 sm:px-6 gap-3'>
        {/* Left: Window Dots & Dynamic Base URL */}
        <div className='flex items-center gap-3'>
          <div className='flex items-center gap-1.5'>
            <span className='size-3 rounded-full bg-rose-500/80' />
            <span className='size-3 rounded-full bg-amber-500/80' />
            <span className='size-3 rounded-full bg-emerald-500/80' />
          </div>
          <div className='flex items-center gap-1.5 rounded-lg border border-border/50 bg-muted/40 px-2.5 py-1 text-xs font-mono text-muted-foreground'>
            <span className='text-foreground/75 font-semibold'>API Base:</span>
            <span className='text-foreground font-mono select-all'>{baseUrl}</span>
            <CopyButton
              value={baseUrl}
              size='sm'
              variant='ghost'
              className='size-6 p-0 hover:bg-muted'
              tooltip={t('Copy Base URL')}
            />
          </div>
        </div>

        {/* Right: Mode Switchers (User Centric) */}
        <div className='flex items-center gap-1 rounded-lg border border-border/60 bg-muted/30 p-1'>
          <button
            type='button'
            onClick={() => setActiveTab('models')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all',
              activeTab === 'models'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Sparkles className='size-3.5 text-amber-500' />
            <span>{t('Explore Models')}</span>
          </button>
          <button
            type='button'
            onClick={() => setActiveTab('apps')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all',
              activeTab === 'apps'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Laptop className='size-3.5 text-emerald-500' />
            <span>{t('1-Click App Setup')}</span>
          </button>
          <button
            type='button'
            onClick={() => setActiveTab('calculator')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all',
              activeTab === 'calculator'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Calculator className='size-3.5 text-blue-500' />
            <span>{t('Cost Estimator')}</span>
          </button>
        </div>
      </div>

      {/* Main Interactive Content */}
      <div className='p-5 sm:p-7'>
        {/* TAB 1: Explore Models & Live Capability Showcase */}
        {activeTab === 'models' && (
          <div className='space-y-6'>
            {/* Top Model Switcher Carousel */}
            <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
              {SHOWCASE_MODELS.map((model, idx) => {
                const isSelected = idx === selectedModelIndex
                return (
                  <button
                    key={model.id}
                    type='button'
                    onClick={() => {
                      setSelectedModelIndex(idx)
                      setSelectedPromptIndex(0)
                    }}
                    className={cn(
                      'flex items-center gap-2.5 rounded-xl border p-3 text-left transition-all',
                      isSelected
                        ? 'border-foreground/30 bg-muted/70 shadow-sm ring-1 ring-border'
                        : 'border-border/60 bg-card/40 hover:border-border hover:bg-muted/30 text-muted-foreground'
                    )}
                  >
                    <div className='flex size-7 shrink-0 items-center justify-center'>
                      {getLobeIcon(model.icon, 20)}
                    </div>
                    <div className='min-w-0'>
                      <div className='truncate text-xs font-bold text-foreground'>
                        {model.name}
                      </div>
                      <div className='truncate text-[10.5px] text-muted-foreground'>
                        {model.context} ctx • {model.speed}
                      </div>
                    </div>
                  </button>
                )
              })}
            </div>

            {/* Model Performance Banner */}
            <div className='flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/60 bg-muted/20 px-4 py-2.5 text-xs'>
              <div className='flex items-center gap-2'>
                <span className='rounded-full bg-emerald-500/10 px-2.5 py-0.5 font-semibold text-emerald-600 dark:text-emerald-400'>
                  {t(currentModel.tagKey)}
                </span>
                <span className='hidden text-muted-foreground sm:inline'>•</span>
                <span className='hidden text-muted-foreground sm:inline'>
                  {t(currentModel.savings)}
                </span>
              </div>
              <div className='flex items-center gap-3 font-mono text-[11px] text-muted-foreground'>
                <span className='flex items-center gap-1'>
                  <Zap className='size-3 text-amber-500' />
                  <strong className='text-foreground'>{currentModel.speed}</strong>
                </span>
                <span>
                  TTFT: <strong className='text-foreground'>{currentModel.latency}</strong>
                </span>
              </div>
            </div>

            {/* Live Prompt & Response Showcase */}
            <div className='rounded-xl border border-border/70 bg-card/80 overflow-hidden shadow-inner'>
              {/* Prompt selection tabs */}
              <div className='flex items-center justify-between border-b border-border/60 bg-muted/40 px-4 py-2'>
                <div className='flex items-center gap-2'>
                  <span className='text-[11px] font-semibold text-muted-foreground uppercase tracking-wider'>
                    {t('Try Sample Prompts')}:
                  </span>
                  <div className='flex items-center gap-1.5'>
                    {currentModel.prompts.map((p, pIdx) => (
                      <button
                        key={p.titleKey}
                        type='button'
                        onClick={() => setSelectedPromptIndex(pIdx)}
                        className={cn(
                          'rounded-md px-2.5 py-1 text-xs transition-colors',
                          selectedPromptIndex === pIdx
                            ? 'bg-background text-foreground font-semibold shadow-xs'
                            : 'text-muted-foreground hover:text-foreground'
                        )}
                      >
                        {t(p.titleKey)}
                      </button>
                    ))}
                  </div>
                </div>

                <div className='hidden sm:flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-mono'>
                  <span className='size-1.5 rounded-full bg-emerald-500 animate-pulse' />
                  <span>{t('Stream Generation Live')}</span>
                </div>
              </div>

              {/* User Prompt Box */}
              <div className='border-b border-border/40 bg-muted/10 p-3.5 sm:p-4 text-xs font-medium text-foreground flex items-start gap-2.5'>
                <div className='mt-0.5 rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-bold text-muted-foreground uppercase'>
                  User
                </div>
                <div className='flex-1 leading-relaxed'>{t(currentPrompt.promptKey)}</div>
              </div>

              {/* AI Response Output Box */}
              <div className='p-4 sm:p-5 text-xs font-mono leading-relaxed bg-background/50 overflow-x-auto max-h-[320px]'>
                {currentPrompt.responseCode ? (
                  <pre className='text-foreground/90'>
                    <code>{currentPrompt.responseCode}</code>
                  </pre>
                ) : (
                  <div className='whitespace-pre-wrap text-foreground/85 font-sans leading-relaxed'>
                    {currentPrompt.responseText}
                  </div>
                )}
              </div>

              {/* Action Bar */}
              <div className='flex flex-wrap items-center justify-between border-t border-border/60 bg-muted/20 px-4 py-2.5 gap-2'>
                <div className='text-[11px] text-muted-foreground'>
                  {t('Compatible with all OpenAI format tools and official client SDKs.')}
                </div>
                <div className='flex items-center gap-2'>
                  <Button
                    size='sm'
                    className='h-8 rounded-lg text-xs font-semibold px-3'
                    render={<Link to='/sign-up' />}
                  >
                    <span>{t('Start Free Trial')}</span>
                    <ArrowRight className='ml-1.5 size-3.5' />
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* TAB 2: 1-Click App Setup Guide */}
        {activeTab === 'apps' && (
          <div className='space-y-6'>
            {/* App Selection Tabs */}
            <div className='flex flex-wrap items-center gap-2 border-b border-border/60 pb-3'>
              {(
                [
                  { id: 'cursor', label: 'Cursor / Cline / Roo', badge: 'AI IDE' },
                  { id: 'cherry', label: 'Cherry Studio', badge: 'Desktop App' },
                  { id: 'nextchat', label: 'NextChat / LibreChat', badge: 'Web UI' },
                  { id: 'python', label: 'Python & Node.js', badge: 'Developer SDK' },
                ] as const
              ).map((app) => (
                <button
                  key={app.id}
                  type='button'
                  onClick={() => setSelectedApp(app.id)}
                  className={cn(
                    'flex items-center gap-2 rounded-xl border px-3.5 py-2 text-xs font-medium transition-all',
                    selectedApp === app.id
                      ? 'border-foreground/30 bg-muted/80 text-foreground font-semibold shadow-xs'
                      : 'border-border/60 bg-card/40 text-muted-foreground hover:border-border hover:bg-muted/30'
                  )}
                >
                  <span>{app.label}</span>
                  <span className='rounded bg-muted px-1.5 py-0.5 text-[9.5px] font-mono text-muted-foreground'>
                    {app.badge}
                  </span>
                </button>
              ))}
            </div>

            {/* Config Panels */}
            {selectedApp === 'cursor' && (
              <div className='grid grid-cols-1 gap-6 lg:grid-cols-2 items-center'>
                <div className='space-y-4'>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('Configure in Cursor in 30 Seconds')}
                  </h3>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Supercharge your Cursor editor with Claude 3.7 Sonnet, DeepSeek R1, or GPT-4o without paying $20/month separate subscriptions.'
                    )}
                  </p>

                  <div className='space-y-3 font-mono text-xs'>
                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        1. Override OpenAI Base URL:
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        2. OpenAI API Key:
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span className='text-muted-foreground'>sk-••••••••••••••••••••••••••••••••</span>
                        <Button
                          size='sm'
                          variant='ghost'
                          className='h-6 px-2 text-[11px] text-foreground'
                          render={<Link to='/keys' />}
                        >
                          {t('Get API Key')}
                        </Button>
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        3. Add Model Names in Cursor:
                      </label>
                      <div className='flex flex-wrap gap-1.5 pt-1'>
                        {['claude-3-7-sonnet', 'deepseek-reasoner', 'deepseek-chat', 'gpt-4o'].map((m) => (
                          <span
                            key={m}
                            className='rounded-md border border-border/60 bg-card px-2 py-1 text-[11px] text-foreground/90 font-mono'
                          >
                            {m}
                          </span>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                {/* Visual Step Illustration Card */}
                <div className='rounded-xl border border-border/70 bg-muted/20 p-5 space-y-3.5'>
                  <div className='text-xs font-bold text-foreground flex items-center gap-2'>
                    <CheckCircle2 className='size-4 text-emerald-500' />
                    <span>{t('Cursor Step-by-Step Setup Guide')}</span>
                  </div>
                  <ol className='space-y-2 text-xs text-muted-foreground leading-relaxed list-decimal list-inside'>
                    <li>{t('Open Cursor Settings (Gear icon) -> Models tab')}</li>
                    <li>{t('Toggle on "Override OpenAI Base URL"')}</li>
                    <li>{t('Paste our Base URL and your personal API Key')}</li>
                    <li>{t('Click Verify to confirm connection, then start coding!')}</li>
                  </ol>
                  <div className='rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-[11.5px] text-emerald-700 dark:text-emerald-300'>
                    💡 {t('Tip: Supports full Composer agent mode, multi-file edits, and tab code completion.')}
                  </div>
                </div>
              </div>
            )}

            {selectedApp === 'cherry' && (
              <div className='grid grid-cols-1 gap-6 lg:grid-cols-2 items-center'>
                <div className='space-y-4'>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('Connect to Cherry Studio Desktop')}
                  </h3>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Cherry Studio is a popular cross-platform desktop AI client. Connect via our OpenAI-compatible endpoint in seconds.'
                    )}
                  </p>

                  <div className='space-y-3 font-mono text-xs'>
                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        API Base URL:
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        Provider Type:
                      </label>
                      <div className='rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-sans text-xs'>
                        OpenAI (or Custom OpenAI Format)
                      </div>
                    </div>
                  </div>
                </div>

                <div className='rounded-xl border border-border/70 bg-muted/20 p-5 space-y-3'>
                  <div className='text-xs font-bold text-foreground flex items-center gap-2'>
                    <CheckCircle2 className='size-4 text-emerald-500' />
                    <span>{t('One-Click Model Synchronization')}</span>
                  </div>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'In Cherry Studio Model Settings, simply click "Check / Sync Models" to automatically load all 50+ available models without manual typing.'
                    )}
                  </p>
                  <div className='flex items-center gap-2 pt-2'>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 rounded-lg text-xs gap-1.5'
                      render={
                        <a href='https://cherry-ai.com' target='_blank' rel='noopener noreferrer' />
                      }
                    >
                      <ExternalLink className='size-3.5' />
                      <span>{t('Download Cherry Studio')}</span>
                    </Button>
                  </div>
                </div>
              </div>
            )}

            {selectedApp === 'nextchat' && (
              <div className='grid grid-cols-1 gap-6 lg:grid-cols-2 items-center'>
                <div className='space-y-4'>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('NextChat & LibreChat Web Setup')}
                  </h3>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Deploy or open NextChat (ChatGPT-Next-Web), enter your API endpoint and key, and enjoy instant fast chat.'
                    )}
                  </p>

                  <div className='space-y-3 font-mono text-xs'>
                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        Endpoint URL:
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>
                  </div>
                </div>

                <div className='rounded-xl border border-border/70 bg-muted/20 p-5 space-y-3'>
                  <div className='text-xs font-bold text-foreground flex items-center gap-2'>
                    <CheckCircle2 className='size-4 text-emerald-500' />
                    <span>{t('Zero CORS or Firewall Issues')}</span>
                  </div>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Our gateway directly serves high-performance CORS headers and SSE stream packaging, allowing smooth browser client connections.'
                    )}
                  </p>
                </div>
              </div>
            )}

            {selectedApp === 'python' && (
              <div className='space-y-3'>
                <div className='flex items-center justify-between'>
                  <div className='text-xs text-muted-foreground'>
                    {t('Drop-in replacement for OpenAI Python & Node.js SDKs with zero codebase refactoring.')}
                  </div>
                  <CopyButton
                    value={`from openai import OpenAI\n\nclient = OpenAI(\n    base_url="${baseUrl}",\n    api_key="sk-your-key-here",\n)\n\nresponse = client.chat.completions.create(\n    model="claude-3-7-sonnet",\n    messages=[{"role": "user", "content": "Hello!"}],\n)\nprint(response.choices[0].message.content)`}
                    size='sm'
                    variant='outline'
                    className='h-7 text-xs rounded-md'
                  />
                </div>
                <div className='rounded-xl border border-border/70 bg-card p-4 font-mono text-xs overflow-x-auto'>
                  <pre className='text-foreground/90'>
                    <code>{`from openai import OpenAI

# 1. Point baseURL to our unified gateway
client = OpenAI(
    base_url="${baseUrl}",
    api_key="sk-your-key-here", # Grab from dashboard
)

# 2. Call any model (Claude 3.7, DeepSeek R1, GPT-4o, Gemini 2.5) seamlessly
response = client.chat.completions.create(
    model="claude-3-7-sonnet",
    messages=[{"role": "user", "content": "Hello from unified API!"}],
)
print(response.choices[0].message.content)`}</code>
                  </pre>
                </div>
              </div>
            )}
          </div>
        )}

        {/* TAB 3: Transparent Pricing & Cost Calculator */}
        {activeTab === 'calculator' && (
          <div className='space-y-6'>
            <div className='grid grid-cols-1 gap-6 lg:grid-cols-12 items-center'>
              {/* Left Column: Interactive Sliders & Options */}
              <div className='space-y-5 lg:col-span-7'>
                <div>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('Pay-As-You-Go Cost Calculator')}
                  </h3>
                  <p className='text-xs text-muted-foreground mt-1'>
                    {t('See how much you save compared to fixed $20/month single-platform subscriptions.')}
                  </p>
                </div>

                {/* Slider for Daily Messages */}
                <div className='space-y-2.5 rounded-xl border border-border/60 bg-muted/20 p-4'>
                  <div className='flex items-center justify-between text-xs'>
                    <span className='font-semibold text-foreground'>
                      {t('Estimated Daily Requests')}:
                    </span>
                    <span className='font-mono font-bold text-foreground text-sm'>
                      {calcDailyMessages} {t('queries / day')}
                    </span>
                  </div>
                  <input
                    type='range'
                    min={10}
                    max={300}
                    step={10}
                    value={calcDailyMessages}
                    onChange={(e) => setCalcDailyMessages(Number.parseInt(e.target.value, 10))}
                    className='w-full accent-foreground cursor-pointer'
                  />
                  <div className='flex justify-between text-[10px] text-muted-foreground font-mono'>
                    <span>10 (Casual)</span>
                    <span>100 (Power User)</span>
                    <span>300+ (Coding & Heavy Dev)</span>
                  </div>
                </div>

                {/* Model Preference Choice */}
                <div className='space-y-2'>
                  <label className='text-xs font-semibold text-foreground'>
                    {t('Primary Model Usage')}:
                  </label>
                  <div className='grid grid-cols-3 gap-2'>
                    {(
                      [
                        { id: 'deepseek', label: 'DeepSeek V3/R1', desc: 'Ultra Value' },
                        { id: 'claude', label: 'Claude 3.7', desc: 'Top Coding' },
                        { id: 'gpt4o', label: 'GPT-4o', desc: 'Flagship Omni' },
                      ] as const
                    ).map((m) => (
                      <button
                        key={m.id}
                        type='button'
                        onClick={() => setCalcModelTier(m.id)}
                        className={cn(
                          'rounded-xl border p-2.5 text-center text-xs transition-all',
                          calcModelTier === m.id
                            ? 'border-foreground/30 bg-muted/80 text-foreground font-bold shadow-xs'
                            : 'border-border/60 bg-card/40 text-muted-foreground hover:bg-muted/30'
                        )}
                      >
                        <div className='font-semibold'>{m.label}</div>
                        <div className='text-[10px] text-muted-foreground'>{m.desc}</div>
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* Right Column: Comparative Results Card */}
              <div className='rounded-2xl border border-border/80 bg-linear-to-b from-card to-muted/30 p-6 lg:col-span-5 space-y-4 shadow-md'>
                <div className='text-xs font-mono uppercase tracking-wider text-muted-foreground'>
                  {t('Estimated Monthly Spend')}
                </div>

                <div className='flex items-baseline gap-2'>
                  <span className='text-4xl font-extrabold text-foreground font-mono tracking-tight'>
                    ${calculatedCost.spend}
                  </span>
                  <span className='text-xs text-muted-foreground'>/ {t('month')}</span>
                </div>

                <div className='space-y-2 border-t border-border/50 pt-3 text-xs'>
                  <div className='flex items-center justify-between text-muted-foreground'>
                    <span>{t('Official Single Subscription')}:</span>
                    <span className='line-through font-mono'>${calculatedCost.official}/mo</span>
                  </div>
                  <div className='flex items-center justify-between font-semibold'>
                    <span className='text-emerald-600 dark:text-emerald-400'>
                      {t('You Save Approximately')}:
                    </span>
                    <span className='rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-600 dark:text-emerald-400 font-mono'>
                      ~{calculatedCost.savings}%
                    </span>
                  </div>
                </div>

                <div className='space-y-1.5 border-t border-border/50 pt-3 text-[11px] text-muted-foreground'>
                  <div className='flex items-center gap-1.5'>
                    <CheckCircle2 className='size-3 text-emerald-500' />
                    <span>{t('Zero monthly commitments — balance never expires')}</span>
                  </div>
                  <div className='flex items-center gap-1.5'>
                    <CheckCircle2 className='size-3 text-emerald-500' />
                    <span>{t('Mix and match 50+ models with a single balance')}</span>
                  </div>
                </div>

                <Button
                  className='w-full h-10 rounded-xl text-xs font-semibold mt-2'
                  render={<Link to='/pricing' />}
                >
                  <span>{t('View Complete Pricing Table')}</span>
                  <ChevronRight className='ml-1.5 size-3.5' />
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
