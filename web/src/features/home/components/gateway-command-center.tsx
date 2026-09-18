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
  Activity,
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  Cpu,
  Layers,
  Lock,
  RefreshCw,
  Server,
  ShieldCheck,
  Terminal,
  Zap,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

type TabType = 'routing' | 'failover' | 'protocol'

interface GatewayCommandCenterProps {
  className?: string
}

export function GatewayCommandCenter({ className }: GatewayCommandCenterProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<TabType>('routing')
  const [selectedClient, setSelectedClient] = useState<string>('cursor')
  const [isFailedOver, setIsFailedOver] = useState<boolean>(false)

  const clients = [
    { id: 'cursor', name: 'Cursor / Cline', type: 'AI IDE' },
    { id: 'cherry', name: 'Cherry Studio', type: 'Desktop' },
    { id: 'python', name: 'OpenAI Python SDK', type: 'Backend' },
    { id: 'nextchat', name: 'NextChat / WebUI', type: 'Web App' },
  ]

  return (
    <div
      className={cn(
        'relative mx-auto w-full max-w-5xl rounded-2xl border border-border/80 bg-card/80 shadow-2xl backdrop-blur-xl transition-all duration-300',
        'dark:border-white/[0.09] dark:bg-[#0b0f17]/90 dark:shadow-[0_20px_60px_-15px_rgba(0,0,0,0.8)]',
        className
      )}
    >
      {/* Chrome Window Header */}
      <div className='flex flex-wrap items-center justify-between border-b border-border/60 px-4 py-3 sm:px-6'>
        {/* Left: Window Dots & Address */}
        <div className='flex items-center gap-3'>
          <div className='flex items-center gap-1.5'>
            <span className='size-3 rounded-full bg-rose-500/80' />
            <span className='size-3 rounded-full bg-amber-500/80' />
            <span className='size-3 rounded-full bg-emerald-500/80' />
          </div>
          <div className='hidden sm:flex items-center gap-2 rounded-md border border-border/50 bg-muted/40 px-3 py-1 text-xs font-mono text-muted-foreground'>
            <Lock className='size-3 text-emerald-500' />
            <span>https://gateway.newapi.pro/v1/routing</span>
          </div>
        </div>

        {/* Center/Right: Tab Switchers */}
        <div className='flex items-center gap-1 rounded-lg border border-border/60 bg-muted/30 p-1'>
          <button
            type='button'
            onClick={() => setActiveTab('routing')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-all',
              activeTab === 'routing'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Activity className='size-3.5 text-emerald-500' />
            <span>{t('Live Traffic Routing')}</span>
          </button>
          <button
            type='button'
            onClick={() => setActiveTab('failover')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-all',
              activeTab === 'failover'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <ShieldCheck className='size-3.5 text-amber-500' />
            <span>{t('Instant Failover')}</span>
          </button>
          <button
            type='button'
            onClick={() => setActiveTab('protocol')}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1 text-xs font-medium transition-all',
              activeTab === 'protocol'
                ? 'bg-background text-foreground shadow-xs font-semibold'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <Layers className='size-3.5 text-blue-500' />
            <span>{t('Protocol Translation')}</span>
          </button>
        </div>
      </div>

      {/* Cockpit Content Body */}
      <div className='p-5 sm:p-7'>
        {/* VIEW 1: Live Traffic Routing */}
        {activeTab === 'routing' && (
          <div className='space-y-6'>
            {/* 3-Column Architecture Visualization */}
            <div className='grid grid-cols-1 gap-4 lg:grid-cols-12 lg:gap-4 items-center'>
              {/* Left Column: Inbound Clients */}
              <div className='space-y-2 lg:col-span-4'>
                <div className='flex items-center justify-between text-xs font-mono text-muted-foreground pb-1 border-b border-border/50'>
                  <span>{t('Inbound Clients')}</span>
                  <span className='text-[10px] uppercase font-semibold text-emerald-500'>● Connected</span>
                </div>
                <div className='grid grid-cols-1 gap-2'>
                  {clients.map((client) => (
                    <button
                      key={client.id}
                      type='button'
                      onClick={() => setSelectedClient(client.id)}
                      className={cn(
                        'flex items-center justify-between rounded-xl border p-2.5 text-left text-xs transition-all',
                        selectedClient === client.id
                          ? 'border-emerald-500/40 bg-emerald-500/10 text-foreground font-semibold shadow-xs'
                          : 'border-border/60 bg-card/40 text-muted-foreground hover:border-border hover:bg-muted/40'
                      )}
                    >
                      <div className='flex items-center gap-2'>
                        <Terminal className='size-3.5 text-muted-foreground' />
                        <span>{client.name}</span>
                      </div>
                      <span className='rounded bg-muted/60 px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground'>
                        {client.type}
                      </span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Center Column: Gateway Core Processing Engine */}
              <div className='space-y-3 lg:col-span-4'>
                <div className='rounded-xl border border-border/80 bg-muted/20 p-4 text-center'>
                  <div className='mx-auto mb-2 flex size-10 items-center justify-center rounded-xl border border-emerald-500/30 bg-emerald-500/10 shadow-xs'>
                    <Cpu className='size-5 text-emerald-500 animate-pulse' />
                  </div>
                  <h4 className='text-xs font-bold font-mono tracking-tight text-foreground'>
                    {t('Gateway Core Engine')}
                  </h4>
                  <p className='text-[11px] text-muted-foreground mt-1'>
                    Zero-Copy Router • ASVS v2 Guard
                  </p>

                  <div className='mt-4 space-y-1.5 text-left'>
                    <div className='flex items-center justify-between rounded-md border border-border/50 bg-background/80 px-2.5 py-1.5 text-[11px] font-mono'>
                      <span className='text-muted-foreground'>Auth & Scope</span>
                      <span className='flex items-center gap-1 text-emerald-500 font-semibold'>
                        <CheckCircle2 className='size-3' /> Verified
                      </span>
                    </div>
                    <div className='flex items-center justify-between rounded-md border border-border/50 bg-background/80 px-2.5 py-1.5 text-[11px] font-mono'>
                      <span className='text-muted-foreground'>Billing Invariant</span>
                      <span className='flex items-center gap-1 text-emerald-500 font-semibold'>
                        <CheckCircle2 className='size-3' /> Protected
                      </span>
                    </div>
                    <div className='flex items-center justify-between rounded-md border border-border/50 bg-background/80 px-2.5 py-1.5 text-[11px] font-mono'>
                      <span className='text-muted-foreground'>Latency P95</span>
                      <span className='font-semibold text-amber-500'>2.4 ms</span>
                    </div>
                  </div>
                </div>
              </div>

              {/* Right Column: Active Upstream Channels */}
              <div className='space-y-2 lg:col-span-4'>
                <div className='flex items-center justify-between text-xs font-mono text-muted-foreground pb-1 border-b border-border/50'>
                  <span>{t('Dispatched Channels')}</span>
                  <span className='text-[10px] text-muted-foreground'>Dynamic Weight</span>
                </div>
                <div className='space-y-2'>
                  <div className='rounded-xl border border-emerald-500/30 bg-card/60 p-2.5'>
                    <div className='flex items-center justify-between text-xs'>
                      <span className='font-semibold text-foreground flex items-center gap-1.5'>
                        <span className='size-2 rounded-full bg-emerald-500' />
                        OpenAI Direct (US-East)
                      </span>
                      <span className='text-[10px] font-mono font-semibold text-emerald-500'>42ms</span>
                    </div>
                    <div className='mt-2 flex items-center justify-between text-[11px] font-mono text-muted-foreground'>
                      <span>{t('Weight')}: 50%</span>
                      <span className='rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400'>
                        {t('Healthy')}
                      </span>
                    </div>
                  </div>

                  <div className='rounded-xl border border-border/60 bg-card/60 p-2.5'>
                    <div className='flex items-center justify-between text-xs'>
                      <span className='font-semibold text-foreground flex items-center gap-1.5'>
                        <span className='size-2 rounded-full bg-emerald-500' />
                        Azure OpenAI (East US)
                      </span>
                      <span className='text-[10px] font-mono font-semibold text-emerald-500'>58ms</span>
                    </div>
                    <div className='mt-2 flex items-center justify-between text-[11px] font-mono text-muted-foreground'>
                      <span>{t('Weight')}: 30%</span>
                      <span className='rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400'>
                        {t('Healthy')}
                      </span>
                    </div>
                  </div>

                  <div className='rounded-xl border border-border/60 bg-card/60 p-2.5'>
                    <div className='flex items-center justify-between text-xs'>
                      <span className='font-semibold text-foreground flex items-center gap-1.5'>
                        <span className='size-2 rounded-full bg-emerald-500' />
                        Claude Sonnet (Bedrock)
                      </span>
                      <span className='text-[10px] font-mono font-semibold text-emerald-500'>78ms</span>
                    </div>
                    <div className='mt-2 flex items-center justify-between text-[11px] font-mono text-muted-foreground'>
                      <span>{t('Weight')}: 20%</span>
                      <span className='rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400'>
                        {t('Healthy')}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* VIEW 2: Instant Failover Simulator */}
        {activeTab === 'failover' && (
          <div className='space-y-5'>
            <div className='flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-muted/20 p-4'>
              <div>
                <h4 className='text-xs font-bold text-foreground'>
                  {t('Simulate upstream provider 502/timeout and watch the gateway automatically reroute traffic within 15ms.')}
                </h4>
                <p className='text-[11px] text-muted-foreground mt-0.5'>
                  {isFailedOver
                    ? t('Traffic Rerouted: 12ms recovery • Zero client errors')
                    : t('Normal Operational State')}
                </p>
              </div>
              <Button
                size='sm'
                variant={isFailedOver ? 'outline' : 'destructive'}
                onClick={() => setIsFailedOver(!isFailedOver)}
                className='h-8 text-xs font-semibold'
              >
                {isFailedOver ? (
                  <>
                    <RefreshCw className='mr-1.5 size-3.5' />
                    <span>{t('Reset Simulation')}</span>
                  </>
                ) : (
                  <>
                    <AlertTriangle className='mr-1.5 size-3.5' />
                    <span>{t('Simulate Outage')}</span>
                  </>
                )}
              </Button>
            </div>

            {/* Channels Failover State Display */}
            <div className='grid grid-cols-1 md:grid-cols-3 gap-3'>
              {/* Channel 1 */}
              <div
                className={cn(
                  'rounded-xl border p-4 transition-all duration-300',
                  isFailedOver
                    ? 'border-rose-500/50 bg-rose-500/10 shadow-xs'
                    : 'border-emerald-500/40 bg-card/60'
                )}
              >
                <div className='flex items-center justify-between'>
                  <span className='text-xs font-bold text-foreground'>Primary OpenAI</span>
                  <span
                    className={cn(
                      'rounded px-1.5 py-0.5 text-[10px] font-mono font-semibold',
                      isFailedOver
                        ? 'bg-rose-500/20 text-rose-600 dark:text-rose-400'
                        : 'bg-emerald-500/10 text-emerald-500'
                    )}
                  >
                    {isFailedOver ? '502 Bad Gateway' : '200 OK'}
                  </span>
                </div>
                <p className='text-[11px] text-muted-foreground mt-2 font-mono'>
                  {isFailedOver
                    ? t('Channel Degraded (502 Bad Gateway)')
                    : t('Primary Channel Active (US-East)')}
                </p>
                <div className='mt-3 flex items-center gap-1.5 text-[10px] font-mono'>
                  <span
                    className={cn(
                      'size-2 rounded-full',
                      isFailedOver ? 'bg-rose-500 animate-ping' : 'bg-emerald-500'
                    )}
                  />
                  <span className={isFailedOver ? 'text-rose-500 font-bold' : 'text-emerald-500'}>
                    {isFailedOver ? t('Degraded') : t('Healthy')}
                  </span>
                </div>
              </div>

              {/* Channel 2 - Backup */}
              <div
                className={cn(
                  'rounded-xl border p-4 transition-all duration-300',
                  isFailedOver
                    ? 'border-emerald-500 bg-emerald-500/10 ring-2 ring-emerald-500/30'
                    : 'border-border/60 bg-card/40'
                )}
              >
                <div className='flex items-center justify-between'>
                  <span className='text-xs font-bold text-foreground'>Azure OpenAI Backup</span>
                  <span
                    className={cn(
                      'rounded px-1.5 py-0.5 text-[10px] font-mono font-semibold',
                      isFailedOver
                        ? 'bg-emerald-500 text-white dark:bg-emerald-500'
                        : 'bg-muted text-muted-foreground'
                    )}
                  >
                    {isFailedOver ? 'Active (Auto-Failover)' : 'Standby'}
                  </span>
                </div>
                <p className='text-[11px] text-muted-foreground mt-2 font-mono'>
                  {t('Backup Channel Standby (Azure HK)')}
                </p>
                <div className='mt-3 flex items-center gap-1.5 text-[10px] font-mono'>
                  <span className='size-2 rounded-full bg-emerald-500' />
                  <span className='text-emerald-500 font-semibold'>
                    {isFailedOver ? t('Active') : t('Healthy')}
                  </span>
                </div>
              </div>

              {/* Channel 3 - Redundant */}
              <div className='rounded-xl border border-border/60 bg-card/40 p-4'>
                <div className='flex items-center justify-between'>
                  <span className='text-xs font-bold text-foreground'>AWS Bedrock</span>
                  <span className='rounded bg-muted px-1.5 py-0.5 text-[10px] font-mono text-muted-foreground'>
                    Standby
                  </span>
                </div>
                <p className='text-[11px] text-muted-foreground mt-2 font-mono'>
                  {t('Third Channel Standby (AWS Bedrock)')}
                </p>
                <div className='mt-3 flex items-center gap-1.5 text-[10px] font-mono'>
                  <span className='size-2 rounded-full bg-emerald-500' />
                  <span className='text-muted-foreground'>{t('Healthy')}</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* VIEW 3: Protocol Translation Matrix */}
        {activeTab === 'protocol' && (
          <div className='space-y-4'>
            <div className='rounded-xl border border-border/70 bg-muted/20 p-4'>
              <h4 className='text-xs font-bold text-foreground'>
                {t('Universal Protocol Translation Matrix')}
              </h4>
              <p className='text-[11px] text-muted-foreground mt-0.5'>
                {t('Any OpenAI SDK, LangChain, LlamaIndex, or desktop agent works natively without modifications.')}
              </p>
            </div>

            <div className='grid grid-cols-1 md:grid-cols-3 gap-3 items-center'>
              <div className='rounded-xl border border-border/70 bg-card/60 p-3.5 space-y-2'>
                <span className='rounded bg-muted/60 px-2 py-0.5 text-[10px] font-mono font-semibold text-muted-foreground'>
                  Inbound Standard
                </span>
                <p className='text-xs font-mono font-semibold text-foreground'>
                  POST /v1/chat/completions
                </p>
                <p className='text-[11px] text-muted-foreground'>
                  OpenAI Chat API standard payload format.
                </p>
              </div>

              <div className='flex flex-col items-center justify-center p-2'>
                <div className='flex items-center gap-2 text-xs font-mono text-emerald-500 font-semibold'>
                  <Zap className='size-4 animate-bounce' />
                  <span>{t('Zero-Copy Translation')}</span>
                  <ArrowRight className='size-3.5' />
                </div>
                <span className='text-[10px] text-muted-foreground mt-1'>
                  &lt; 0.2ms translation overhead
                </span>
              </div>

              <div className='rounded-xl border border-border/70 bg-card/60 p-3.5 space-y-2'>
                <span className='rounded bg-muted/60 px-2 py-0.5 text-[10px] font-mono font-semibold text-muted-foreground'>
                  Outbound Native
                </span>
                <p className='text-xs font-mono font-semibold text-foreground'>
                  Claude / Gemini / DeepSeek APIs
                </p>
                <p className='text-[11px] text-muted-foreground'>
                  Auto-mapped parameters, tool calls & SSE streaming.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Bottom Cockpit Status Footer */}
        <div className='mt-6 pt-4 border-t border-border/50 flex flex-wrap items-center justify-between gap-3 text-[11px] font-mono text-muted-foreground'>
          <div className='flex items-center gap-4'>
            <span className='flex items-center gap-1.5'>
              <Server className='size-3.5 text-emerald-500' />
              <span>40+ LLM Backends</span>
            </span>
            <span className='flex items-center gap-1.5'>
              <Zap className='size-3.5 text-amber-500' />
              <span>Sub-5ms Gateway Latency</span>
            </span>
            <span className='hidden sm:flex items-center gap-1.5'>
              <ShieldCheck className='size-3.5 text-blue-500' />
              <span>ASVS & Session Hardened</span>
            </span>
          </div>
          <span className='text-emerald-600 dark:text-emerald-400 font-semibold'>
            100% Drop-In Compatible
          </span>
        </div>
      </div>
    </div>
  )
}
