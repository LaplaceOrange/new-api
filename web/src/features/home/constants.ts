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
/**
 * Home page constants
 * All structured data for home page sections
 */
import type { TFunction } from 'i18next'

export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

// Supported Upstream AI Providers
export interface UpstreamProviderItem {
  id: string
  name: string
  icon: string
  highlightModel: string
  tags?: string[]
}

export const UPSTREAM_PROVIDERS: readonly UpstreamProviderItem[] = [
  { id: 'openai', name: 'OpenAI', icon: 'OpenAI', highlightModel: 'GPT-4.5 / o3-mini' },
  { id: 'claude', name: 'Anthropic', icon: 'Claude.Color', highlightModel: 'Claude 3.7 Sonnet' },
  { id: 'gemini', name: 'Google', icon: 'Gemini.Color', highlightModel: 'Gemini 2.5 Pro' },
  { id: 'deepseek', name: 'DeepSeek', icon: 'DeepSeek.Color', highlightModel: 'DeepSeek V3 / R1' },
  { id: 'qwen', name: 'Alibaba Qwen', icon: 'Qwen.Color', highlightModel: 'Qwen 2.5 Max' },
  { id: 'meta', name: 'Meta AI', icon: 'Meta.Color', highlightModel: 'Llama 3.3 70B' },
  { id: 'mistral', name: 'Mistral AI', icon: 'Mistral.Color', highlightModel: 'Mistral Large 2' },
  { id: 'grok', name: 'xAI Grok', icon: 'Grok.Color', highlightModel: 'Grok 3 Beta' },
  { id: 'ollama', name: 'Ollama', icon: 'Ollama', highlightModel: 'Local & Self-Hosted' },
  { id: 'midjourney', name: 'Midjourney', icon: 'Midjourney', highlightModel: 'Imagine / Fast' },
] as const

// Supported Downstream Applications
export interface SupportedAppItem {
  name: string
  icon?: string
  badge: string
  url?: string
}

export const SUPPORTED_APPS: readonly SupportedAppItem[] = [
  { name: 'Cherry Studio', icon: 'CherryStudio.Color', badge: 'Desktop Chat', url: 'https://cherry-ai.com' },
  { name: 'CC Switch', badge: 'Multi-Protocol Hub', url: 'https://ccswitch.io' },
  { name: 'Open WebUI', icon: 'OpenWebUI.Color', badge: 'Self-Hosted WebUI' },
  { name: 'Dify', icon: 'Dify.Color', badge: 'AI Agent & Workflow' },
  { name: 'Cursor / Cline', badge: 'AI IDE & Agent' },
  { name: 'NextChat', badge: 'Cross-Platform Chat' },
  { name: 'LibreChat', badge: 'Enterprise AI UI' },
] as const

// Hero API Protocol Demos
export type AccentTone = 'emerald' | 'amber' | 'blue' | 'violet'

export interface ApiDemoConfig {
  id: string
  label: string
  protocolBadge: string
  method: 'POST' | 'GET'
  endpoint: string
  headers: string[]
  request: string[]
  responseStream: string[]
  tokens: number
  latency: number
  costEstimate: string
  accent: AccentTone
}

export const API_DEMOS: ApiDemoConfig[] = [
  {
    id: 'openai-chat',
    label: 'OpenAI Chat',
    protocolBadge: 'OpenAI Protocol',
    method: 'POST',
    endpoint: '/v1/chat/completions',
    headers: ['"Authorization: Bearer sk-gateway-••••"'],
    request: [
      '"model": "gpt-4.5-preview",',
      '"stream": true,',
      '"messages": [',
      '  { "role": "user", "content": "Explain AI gateway benefits." }',
      ']',
    ],
    responseStream: [
      'A unified AI gateway simplifies integrations, prevents vendor lock-in,',
      'and provides high-availability failover with token-exact billing.',
    ],
    tokens: 42,
    latency: 84,
    costEstimate: '$0.00012',
    accent: 'emerald',
  },
  {
    id: 'claude-messages',
    label: 'Claude Messages',
    protocolBadge: 'Anthropic Protocol',
    method: 'POST',
    endpoint: '/v1/messages',
    headers: ['"x-api-key: sk-gateway-••••"', '"anthropic-version: 2023-06-01"'],
    request: [
      '"model": "claude-3-7-sonnet",',
      '"max_tokens": 1024,',
      '"messages": [',
      '  { "role": "user", "content": "Deploy smart load balancing." }',
      ']',
    ],
    responseStream: [
      'Traffic dynamically routes across active upstream endpoints.',
      'Circuit breakers prevent cascade outages with automatic health recovery.',
    ],
    tokens: 38,
    latency: 96,
    costEstimate: '$0.00011',
    accent: 'blue',
  },
  {
    id: 'gemini-content',
    label: 'Gemini Content',
    protocolBadge: 'Google GenAI Protocol',
    method: 'POST',
    endpoint: '/v1beta/models/gemini-2.5-pro:streamGenerateContent',
    headers: ['"x-goog-api-key: sk-gateway-••••"'],
    request: [
      '"contents": [',
      '  { "role": "user",',
      '    "parts": [{ "text": "Analyze gateway throughput." }] }',
      ']',
    ],
    responseStream: [
      'Sub-5ms proxy latency with zero-copy stream chunking.',
      'Supports high-concurrency SSE connections under heavy workloads.',
    ],
    tokens: 35,
    latency: 78,
    costEstimate: '$0.00007',
    accent: 'violet',
  },
  {
    id: 'deepseek-r1',
    label: 'DeepSeek R1',
    protocolBadge: 'Reasoning Protocol',
    method: 'POST',
    endpoint: '/v1/chat/completions',
    headers: ['"Authorization: Bearer sk-gateway-••••"'],
    request: [
      '"model": "deepseek-reasoner",',
      '"stream": true,',
      '"messages": [',
      '  { "role": "user", "content": "Evaluate multi-channel failover." }',
      ']',
    ],
    responseStream: [
      '<think>Evaluating channel weights, health latency, and error quotas.</think>',
      'Optimal route selected: 100% success rate across distributed regions.',
    ],
    tokens: 64,
    latency: 110,
    costEstimate: '$0.00009',
    accent: 'amber',
  },
]

// Stats Section Data
export interface StatMetric {
  value: number
  prefix?: string
  suffix?: string
  decimals?: number
  labelKey: string
  sublabelKey: string
}

export const STAT_METRICS: readonly StatMetric[] = [
  {
    value: 50,
    suffix: '+',
    labelKey: 'Frontier AI Models Supported',
    sublabelKey: 'Covering GPT-4o, Claude 3.7, DeepSeek & Gemini',
  },
  {
    value: 99.99,
    suffix: '%',
    decimals: 2,
    labelKey: 'Service Availability SLA',
    sublabelKey: 'Multi-region redundancy & zero-downtime routing',
  },
  {
    value: 200,
    prefix: '< ',
    suffix: 'ms',
    labelKey: 'Lightning TTFT Latency',
    sublabelKey: 'Direct accelerated lines with minimal wait time',
  },
  {
    value: 100,
    suffix: '%',
    labelKey: 'OpenAI Protocol Compatible',
    sublabelKey: 'Drop-in integration for Cursor, Cherry Studio & more',
  },
] as const


// Quickstart Code Snippets
export interface QuickstartSnippet {
  id: string
  label: string
  language: string
  code: string
}

export const QUICKSTART_SNIPPETS: QuickstartSnippet[] = [
  {
    id: 'curl',
    label: 'cURL',
    language: 'bash',
    code: `curl https://api.yourdomain.com/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer sk-your-gateway-key" \\
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Hello from New API!"}]
  }'`,
  },
  {
    id: 'python',
    label: 'Python (OpenAI SDK)',
    language: 'python',
    code: `from openai import OpenAI

# Simply redirect baseURL to your New API instance
client = OpenAI(
    api_key="sk-your-gateway-key",
    base_url="https://api.yourdomain.com/v1"
)

response = client.chat.completions.create(
    model="claude-3-7-sonnet", # Works with any provider!
    messages=[{"role": "user", "content": "Hello AI Gateway!"}]
)
print(response.choices[0].message.content)`,
  },
  {
    id: 'typescript',
    label: 'TypeScript / Node.js',
    language: 'typescript',
    code: `import OpenAI from 'openai';

// Compatible with official OpenAI SDK
const openai = new OpenAI({
  apiKey: process.env.NEW_API_KEY,
  baseURL: 'https://api.yourdomain.com/v1',
});

const stream = await openai.chat.completions.create({
  model: 'deepseek-chat',
  messages: [{ role: 'user', content: 'Streaming test' }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}`,
  },
]

export const GATEWAY_FEATURES = [
  'Cost Tracking',
  'Model Access',
  'Guardrails',
  'Observability',
  'Budgets',
  'Load Balancing',
  'Rate Limiting',
  'Token Mgmt',
  'Prompt Caching',
  'Pass-Through',
] as const

export function getGatewayFeatures(t: TFunction): string[] {
  return GATEWAY_FEATURES.map((feature) => t(feature))
}

export function getStatsWithTranslation(t: TFunction) {
  return STAT_METRICS.map((stat) => ({
    ...stat,
    label: t(stat.labelKey),
    sublabel: t(stat.sublabelKey),
  }))
}
