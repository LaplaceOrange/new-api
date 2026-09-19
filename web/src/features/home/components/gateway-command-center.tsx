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
import { CheckCircle2, ExternalLink } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

interface GatewayCommandCenterProps {
  className?: string
}

interface ShowcaseModel {
  id: string
  name: string
  icon: string
  speed: string
  context: string
}

const SHOWCASE_MODELS: ShowcaseModel[] = [
  {
    id: 'deepseek-v4-1',
    name: 'DeepSeek V4.1 Flash',
    icon: 'DeepSeek.Color',
    speed: '145 tok/s',
    context: '1M',
  },
  {
    id: 'gpt-6-astra',
    name: 'GPT 6 Astra',
    icon: 'OpenAI',
    speed: '120 tok/s',
    context: '1.05M',
  },
  {
    id: 'gemini-3-8-flash',
    name: 'Gemini 3.8 Flash',
    icon: 'Gemini.Color',
    speed: '160 tok/s',
    context: '1M',
  },
  {
    id: 'grok-4-6',
    name: 'Grok 4.6',
    icon: 'Grok.Color',
    speed: '130 tok/s',
    context: '500K',
  },
]

export function GatewayCommandCenter({ className }: GatewayCommandCenterProps) {
  const { t } = useTranslation()
  const [selectedApp, setSelectedApp] = useState<'codex' | 'cursor' | 'cherry' | 'python'>('codex')

  const baseUrl = useMemo(() => {
    if (typeof window !== 'undefined' && window.location.origin) {
      return `${window.location.origin}/v1`
    }
    return 'https://api.yourdomain.com/v1'
  }, [])

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
            <span className='text-foreground/75 font-semibold'>{t('API Base:')}</span>
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
      </div>

      {/* Main Interactive Content */}
      <div className='p-5 sm:p-7'>
        <div className='space-y-6'>
            <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
              {SHOWCASE_MODELS.map((model) => (
                <div
                  key={model.id}
                  className='flex items-center gap-2.5 rounded-xl border border-border/60 bg-card/40 p-3'
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
                </div>
              ))}
            </div>

            {/* App Selection Tabs */}
            <div className='flex flex-wrap items-center gap-2 border-b border-border/60 pb-3'>
              {(
                [
                  { id: 'codex', label: 'Codex', badge: 'AI Coding Agent' },
                  { id: 'cursor', label: 'Cursor / Cline / Roo', badge: 'AI IDE' },
                  { id: 'cherry', label: 'Cherry Studio', badge: 'Desktop App' },
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
                  <span>{t(app.label)}</span>
                  <span className='rounded bg-muted px-1.5 py-0.5 text-[9.5px] font-mono text-muted-foreground'>
                    {t(app.badge)}
                  </span>
                </button>
              ))}
            </div>

            {/* Config Panels */}
            {selectedApp === 'codex' && (
              <div className='grid grid-cols-1 gap-6 lg:grid-cols-2 items-center'>
                <div className='space-y-4'>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('Configure Codex in Minutes')}
                  </h3>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Connect Codex through CC Switch. Copy the Base URL, create an API key, add the models, then start using Codex.'
                    )}
                  </p>

                  <div className='space-y-3 font-mono text-xs'>
                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('1. Base URL:')}
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('2. API Key:')}
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
                        {t('3. Add models:')}
                      </label>
                      <div className='flex flex-wrap gap-1.5 pt-1'>
                        {['gpt-5.6-terra', 'gpt-5.6-sol', 'gpt-6-astra'].map((m) => (
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

                <div className='rounded-xl border border-border/70 bg-muted/20 p-5 space-y-3.5'>
                  <div className='text-xs font-bold text-foreground flex items-center gap-2'>
                    <CheckCircle2 className='size-4 text-emerald-500' />
                    <span>{t('Codex Step-by-Step Setup Guide')}</span>
                  </div>
                  <ol className='space-y-2 text-xs text-muted-foreground leading-relaxed list-decimal list-inside'>
                    <li>{t('Download and install CC Switch from ccswitch.io')}</li>
                    <li>{t('Add a new provider in CC Switch')}</li>
                    <li>{t('Paste the Base URL shown on the left')}</li>
                    <li>{t('Create an API key and paste it into the API key field')}</li>
                    <li>{t('Add models gpt-5.6-terra, gpt-5.6-sol, and gpt-6-astra')}</li>
                    <li>{t('Save the provider, start Codex, and begin using it')}</li>
                  </ol>
                  <div className='flex items-center gap-2 pt-1'>
                    <Button
                      size='sm'
                      variant='outline'
                      className='h-8 rounded-lg text-xs gap-1.5'
                      render={
                        <a href='https://ccswitch.io' target='_blank' rel='noopener noreferrer' />
                      }
                    >
                      <ExternalLink className='size-3.5' />
                      <span>{t('Download CC Switch')}</span>
                    </Button>
                  </div>
                  <div className='rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-[11.5px] text-emerald-700 dark:text-emerald-300'>
                    💡 {t('Tip: After saving, launch Codex and it will use this provider automatically.')}
                  </div>
                </div>
              </div>
            )}

            {selectedApp === 'cursor' && (
              <div className='grid grid-cols-1 gap-6 lg:grid-cols-2 items-center'>
                <div className='space-y-4'>
                  <h3 className='text-base font-bold text-foreground'>
                    {t('Configure in Cursor in 30 Seconds')}
                  </h3>
                  <p className='text-xs text-muted-foreground leading-relaxed'>
                    {t(
                      'Supercharge your Cursor editor with GPT 6 Astra, DeepSeek V4.1 Flash, Gemini 3.8 Flash, or Grok 4.6 without paying $20/month separate subscriptions.'
                    )}
                  </p>

                  <div className='space-y-3 font-mono text-xs'>
                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('1. Override OpenAI Base URL:')}
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('2. OpenAI API Key:')}
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
                        {t('3. Add Model Names in Cursor:')}
                      </label>
                      <div className='flex flex-wrap gap-1.5 pt-1'>
                        {['deepseek-v4-1', 'gpt-6-astra', 'gemini-3-8-flash', 'grok-4-6'].map((m) => (
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
                        {t('1. Provider Type:')}
                      </label>
                      <div className='rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-sans text-xs'>
                        {t('OpenAI (or Custom OpenAI Format)')}
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('2. Base URL:')}
                      </label>
                      <div className='flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2 text-foreground font-mono'>
                        <span>{baseUrl}</span>
                        <CopyButton value={baseUrl} size='sm' variant='ghost' className='size-6 p-0' />
                      </div>
                    </div>

                    <div className='space-y-1.5'>
                      <label className='text-muted-foreground font-sans font-medium text-[11px]'>
                        {t('3. API Key:')}
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

            {selectedApp === 'python' && (
              <div className='space-y-3'>
                <div className='flex items-center justify-between'>
                  <div className='text-xs text-muted-foreground'>
                    {t('Drop-in replacement for OpenAI Python & Node.js SDKs with zero codebase refactoring.')}
                  </div>
                  <CopyButton
                    value={`from openai import OpenAI\n\nclient = OpenAI(\n    base_url="${baseUrl}",\n    api_key="sk-your-key-here",\n)\n\nresponse = client.chat.completions.create(\n    model="gpt-6-astra",\n    messages=[{"role": "user", "content": "Hello!"}],\n)\nprint(response.choices[0].message.content)`}
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

# 2. Call any model (GPT 6 Astra, DeepSeek V4.1 Flash, Gemini 3.8 Flash, Grok 4.6) seamlessly
response = client.chat.completions.create(
    model="gpt-6-astra",
    messages=[{"role": "user", "content": "Hello from unified API!"}],
)
print(response.choices[0].message.content)`}</code>
                  </pre>
                </div>
              </div>
            )}
          </div>
      </div>
    </div>
  )
}
