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
import { Play, RotateCw } from 'lucide-react'
import {
  useState,
  useEffect,
  useRef,
  useCallback,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { cn } from '@/lib/utils'

import { API_DEMOS, type AccentTone } from '../constants'

const ACCENT_CLASSES: Record<
  AccentTone,
  {
    activeText: string
    activeBorder: string
    badge: string
  }
> = {
  emerald: {
    activeText: 'text-emerald-600 dark:text-emerald-400',
    activeBorder: 'border-emerald-500 dark:border-emerald-400',
    badge:
      'bg-emerald-500/10 text-emerald-600 dark:bg-emerald-400/10 dark:text-emerald-400',
  },
  amber: {
    activeText: 'text-amber-600 dark:text-amber-400',
    activeBorder: 'border-amber-500 dark:border-amber-400',
    badge:
      'bg-amber-500/10 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400',
  },
  blue: {
    activeText: 'text-blue-600 dark:text-blue-400',
    activeBorder: 'border-blue-500 dark:border-blue-400',
    badge:
      'bg-blue-500/10 text-blue-600 dark:bg-blue-400/10 dark:text-blue-400',
  },
  violet: {
    activeText: 'text-violet-600 dark:text-violet-400',
    activeBorder: 'border-violet-500 dark:border-violet-400',
    badge:
      'bg-violet-500/10 text-violet-600 dark:bg-violet-400/10 dark:text-violet-400',
  },
}

const CYCLE_INTERVAL = 6000

interface HeroTerminalDemoProps {
  className?: string
}

export function HeroTerminalDemo(props: HeroTerminalDemoProps) {
  const { t } = useTranslation()
  const [activeIndex, setActiveIndex] = useState(0)
  const [streamProgress, setStreamProgress] = useState(0)
  const [isStreaming, setIsStreaming] = useState(false)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const streamTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const demo = API_DEMOS[activeIndex]
  const fullResponse = demo.responseStream.join(' ')
  const accent = ACCENT_CLASSES[demo.accent]

  // Simulate streaming output token by token
  const startStream = useCallback((text: string) => {
    if (streamTimerRef.current) clearInterval(streamTimerRef.current)
    setStreamProgress(0)
    setIsStreaming(true)

    const totalChars = text.length
    let current = 0
    const step = 4

    streamTimerRef.current = setInterval(() => {
      current += step
      if (current >= totalChars) {
        setStreamProgress(totalChars)
        setIsStreaming(false)
        if (streamTimerRef.current) clearInterval(streamTimerRef.current)
      } else {
        setStreamProgress(current)
      }
    }, 28)
  }, [])

  // Start stream when active demo changes
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    if (mq.matches) {
      setStreamProgress(fullResponse.length)
      setIsStreaming(false)
      return
    }

    startStream(fullResponse)

    return () => {
      if (streamTimerRef.current) clearInterval(streamTimerRef.current)
    }
  }, [activeIndex, fullResponse, startStream])

  // Auto cycle tabs
  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    if (mq.matches) return

    intervalRef.current = setInterval(() => {
      setActiveIndex((prev) => (prev + 1) % API_DEMOS.length)
    }, CYCLE_INTERVAL)

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  const handleSelect = (index: number) => {
    if (index === activeIndex) return
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
      intervalRef.current = null
    }
    setActiveIndex(index)
  }

  const handleReplay = () => {
    startStream(fullResponse)
  }

  // Generate complete curl string for one-click copy
  const curlString = [
    `curl -X ${demo.method} "https://api.yourdomain.com${demo.endpoint}" \\`,
    ...demo.headers.map((h) => `  -H ${h} \\`),
    `  -d '{`,
    ...demo.request.map((l) => `    ${l}`),
    `  }'`,
  ].join('\n')

  return (
    <div className={cn('mx-auto w-full max-w-2xl', props.className)}>
      <div
        className={cn(
          'overflow-hidden rounded-2xl border backdrop-blur-md transition-all duration-300',
          'border-border/70 bg-card/95 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.12)]',
          'dark:border-white/[0.08] dark:bg-[#0c1017]/95 dark:shadow-[0_25px_65px_-20px_rgba(0,0,0,0.7)]'
        )}
      >
        {/* Terminal Header & Tab Strip */}
        <div
          className={cn(
            'flex items-center justify-between border-b px-3 sm:px-4',
            'border-border/60 bg-muted/30 dark:border-white/[0.06] dark:bg-white/[0.02]'
          )}
        >
          {/* Tabs */}
          <div className='flex items-center gap-1 sm:gap-2 overflow-x-auto no-scrollbar'>
            {API_DEMOS.map((item, index) => {
              const tone = ACCENT_CLASSES[item.accent]
              const isActive = index === activeIndex
              return (
                <button
                  key={item.id}
                  type='button'
                  onClick={() => handleSelect(index)}
                  className={cn(
                    'relative -mb-px flex items-center gap-1.5 border-b-2 px-2.5 py-3 text-xs font-semibold tracking-wide transition-all sm:px-3',
                    isActive
                      ? `${tone.activeBorder} ${tone.activeText}`
                      : 'text-muted-foreground/70 hover:text-foreground border-transparent'
                  )}
                >
                  <span>{item.label}</span>
                </button>
              )
            })}
          </div>

          {/* Right Status Badge & Actions */}
          <div className='flex items-center gap-2 pl-2'>
            <button
              type='button'
              onClick={handleReplay}
              title={t('Replay stream')}
              aria-label={t('Replay stream')}
              className='text-muted-foreground hover:text-foreground hover:bg-muted/80 flex size-7 items-center justify-center rounded-md transition-colors'
            >
              {isStreaming ? (
                <RotateCw className='size-3.5 animate-spin text-blue-500' />
              ) : (
                <Play className='size-3.5 fill-current' />
              )}
            </button>
            <div className='flex items-center gap-1.5 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2.5 py-1 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400'>
              <span className='size-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)] animate-pulse' />
              <span>200 OK</span>
            </div>
          </div>
        </div>

        {/* Endpoint Row with Copy Button */}
        <div
          className={cn(
            'flex items-center justify-between border-b px-4 py-2.5 text-xs',
            'border-border/50 bg-background/50 dark:border-white/[0.04]'
          )}
        >
          <div className='flex items-center gap-2 overflow-hidden'>
            <span
              className={cn(
                'rounded-md px-1.5 py-0.5 font-mono text-[10px] font-bold tracking-wider',
                accent.badge
              )}
            >
              {demo.method}
            </span>
            <code className='text-foreground/80 font-mono text-[12px] truncate'>
              {demo.endpoint}
            </code>
          </div>
          <div className='flex items-center gap-1.5 shrink-0'>
            <span className='text-muted-foreground/60 hidden text-[10px] font-mono sm:inline-block'>
              {demo.protocolBadge}
            </span>
            <CopyButton
              value={curlString}
              size='sm'
              variant='ghost'
              className='size-7 p-0'
              tooltip={t('Copy cURL')}
            />
          </div>
        </div>

        {/* Request & Response Split Panel */}
        <div className='grid font-mono text-[12px] leading-[1.55]'>
          {/* Request Section */}
          <div className='border-b border-border/40 px-4 py-3.5 dark:border-white/[0.04]'>
            <div className='text-muted-foreground/60 mb-2 flex items-center justify-between text-[10px] font-sans font-bold tracking-widest uppercase'>
              <span>{t('Request')}</span>
              <span className='text-muted-foreground/50'>JSON / POST</span>
            </div>
            <div className='space-y-0.5 text-foreground/85'>
              <div className='text-muted-foreground/80'>
                <span className='text-emerald-600 dark:text-emerald-400 font-medium'>curl</span>{' '}
                <span className='text-blue-600 dark:text-blue-400'>-X POST</span>{' '}
                &quot;{demo.endpoint}&quot; \
              </div>
              {demo.headers.map((h) => (
                <div key={h} className='pl-3 text-muted-foreground/75'>
                  <span className='text-blue-600 dark:text-blue-400'>-H</span> {h} \
                </div>
              ))}
              <div className='pl-3 text-amber-700 dark:text-amber-300'>
                <span className='text-blue-600 dark:text-blue-400'>-d</span> &apos;&#123;
              </div>
              {demo.request.map((line) => (
                <div key={line} className='pl-6 text-foreground/80'>
                  {renderFormattedJson(line)}
                </div>
              ))}
              <div className='pl-3 text-amber-700 dark:text-amber-300'>&#125;&apos;</div>
            </div>
          </div>

          {/* Real-time Streaming Response Section */}
          <div className='border-border/40 bg-muted/20 px-4 py-3.5 dark:border-white/[0.04] dark:bg-white/[0.015]'>
            <div className='text-muted-foreground/60 mb-2 flex items-center justify-between text-[10px] font-sans font-bold tracking-widest uppercase'>
              <span className='flex items-center gap-1.5'>
                <span>{t('Streaming Response')}</span>
                {isStreaming && (
                  <span className='size-1.5 rounded-full bg-blue-500 animate-ping' />
                )}
              </span>
              <span className='text-emerald-600 dark:text-emerald-400 font-semibold'>
                {demo.tokens} tokens
              </span>
            </div>
            <div className='rounded-lg border border-border/40 bg-background/60 p-3 text-foreground/90 shadow-xs dark:border-white/[0.04] dark:bg-background/40'>
              <span className='text-foreground/90'>
                {fullResponse.slice(0, streamProgress)}
              </span>
              {isStreaming && (
                <span className='inline-block w-1.5 h-3.5 ml-0.5 align-middle bg-blue-500 animate-pulse' />
              )}
            </div>
          </div>
        </div>

        {/* Footer Metrics */}
        <div
          className={cn(
            'flex items-center justify-between border-t px-4 py-2.5 text-[11px] tabular-nums',
            'border-border/50 bg-muted/40 dark:border-white/[0.05] dark:bg-white/[0.02]'
          )}
        >
          <div className='flex items-center gap-3 text-muted-foreground'>
            <span className='flex items-center gap-1'>
              <span className='font-mono font-semibold text-foreground'>{demo.latency}</span>
              <span className='text-[10px] uppercase'>ms latency</span>
            </span>
            <span className='size-1 rounded-full bg-foreground/20' />
            <span className='flex items-center gap-1'>
              <span className='font-mono font-semibold text-foreground'>{demo.tokens}</span>
              <span className='text-[10px] uppercase'>tokens</span>
            </span>
            <span className='size-1 rounded-full bg-foreground/20' />
            <span className='flex items-center gap-1'>
              <span className='font-mono font-semibold text-foreground'>{demo.costEstimate}</span>
              <span className='text-[10px] uppercase'>est. cost</span>
            </span>
          </div>
          <span className='text-muted-foreground/60 font-mono text-[10px] tracking-wider uppercase hidden sm:inline-block'>
            SSE Chunked · Zero-Copy
          </span>
        </div>
      </div>
    </div>
  )
}

function renderFormattedJson(line: string): ReactNode {
  const parts = line.split(':')
  if (parts.length >= 2) {
    const key = parts[0]
    const val = parts.slice(1).join(':')
    return (
      <>
        <span className='text-sky-700 dark:text-sky-300 font-medium'>{key}</span>:
        <span className='text-amber-700 dark:text-amber-300'>{val}</span>
      </>
    )
  }
  return <span className='text-foreground/75'>{line}</span>
}
