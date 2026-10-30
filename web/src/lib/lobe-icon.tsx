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
 * LobeHub Icon Loader
 * Dynamically load and render icons from @lobehub/icons
 *
 * Supports:
 * - Basic: "OpenAI", "OpenAI.Color"
 * - Chained properties: "OpenAI.Avatar.type={'platform'}"
 * - Size parameter: getLobeIcon("OpenAI", 20)
 */
import CherryStudioColor from '@lobehub/icons/es/CherryStudio/components/Color'
import CodexColor from '@lobehub/icons/es/Codex/components/Color'
import DeepSeekColor from '@lobehub/icons/es/DeepSeek/components/Color'
import DifyColor from '@lobehub/icons/es/Dify/components/Color'
import GeminiColor from '@lobehub/icons/es/Gemini/components/Color'
import Grok from '@lobehub/icons/es/Grok/components/Mono'
import OpenAI from '@lobehub/icons/es/OpenAI/components/Mono'
import OpenWebUI from '@lobehub/icons/es/OpenWebUI/components/Mono'
import QwenColor from '@lobehub/icons/es/Qwen/components/Color'
import { toc } from '@lobehub/icons/es/toc'
import XiaomiMiMo from '@lobehub/icons/es/XiaomiMiMo/components/Mono'
import ZhipuColor from '@lobehub/icons/es/Zhipu/components/Color'
import { lazy, Suspense, type ComponentType, type ReactNode } from 'react'

import { IconSub2api } from '@/assets/custom/icon-sub2api'

const CUSTOM_ICONS: Record<string, ComponentType<{ size?: number }>> = {
  Sub2API: IconSub2api,
  WorkBuddy: (props) => (
    <img
      src='https://raw.githubusercontent.com/DGZSbot/ai-icon/refs/heads/main/WorkBuddy.png'
      alt=''
      width={props.size ?? 20}
      height={props.size ?? 20}
      className='shrink-0 object-contain'
      referrerPolicy='no-referrer'
    />
  ),
}

// Homepage icons use SVG leaves so Avatar/Combine dependencies stay asynchronous.
const COMMON_ICONS = {
  OpenAI,
  'DeepSeek.Color': DeepSeekColor,
  'Gemini.Color': GeminiColor,
  Grok,
  'Qwen.Color': QwenColor,
  XiaomiMiMo,
  'Zhipu.Color': ZhipuColor,
  'CherryStudio.Color': CherryStudioColor,
  'Codex.Color': CodexColor,
  OpenWebUI,
  'Dify.Color': DifyColor,
}
const iconCatalog = new Map(toc.map((icon) => [icon.id, icon]))
const lazyIcons = new Map<string, ComponentType<Record<string, unknown>>>()

/**
 * Parse a property value from string to appropriate type
 * @param raw - Raw string value
 * @returns Parsed value (boolean, number, or string)
 */
function parseValue(raw: string | undefined | null): string | number | boolean {
  if (raw == null) return true

  let v = String(raw).trim()

  // Remove curly braces
  if (v.startsWith('{') && v.endsWith('}')) {
    v = v.slice(1, -1).trim()
  }

  // Remove quotes
  if (
    (v.startsWith('"') && v.endsWith('"')) ||
    (v.startsWith("'") && v.endsWith("'"))
  ) {
    return v.slice(1, -1)
  }

  // Boolean
  if (v === 'true') return true
  if (v === 'false') return false

  // Number
  if (/^-?\d+(?:\.\d+)?$/.test(v)) return Number(v)

  // Return as string
  return v
}

/**
 * Get LobeHub icon component by name
 * @param iconName - Icon name/description (e.g., "OpenAI", "OpenAI.Color", "Claude.Avatar")
 * @param size - Icon size (default: 20)
 * @returns Icon component or fallback
 *
 * @example
 * getLobeIcon("OpenAI", 24)
 * getLobeIcon("OpenAI.Color", 20)
 * getLobeIcon("Claude.Avatar.type={'platform'}", 32)
 */
export function getLobeIcon(
  iconName: string | undefined | null,
  size: number = 20
): ReactNode {
  if (!iconName || typeof iconName !== 'string') {
    return (
      <div
        className='bg-muted text-muted-foreground flex items-center justify-center rounded-full text-xs font-medium'
        style={{ width: size, height: size }}
      >
        ?
      </div>
    )
  }

  const trimmedName = iconName.trim()
  if (!trimmedName) {
    return (
      <div
        className='bg-muted text-muted-foreground flex items-center justify-center rounded-full text-xs font-medium'
        style={{ width: size, height: size }}
      >
        ?
      </div>
    )
  }

  // Parse component path and chained properties
  const segments = trimmedName.split('.')
  const baseKey = segments[0]
  const CustomIcon = CUSTOM_ICONS[baseKey]
  if (CustomIcon) {
    return <CustomIcon size={size} />
  }
  const fallback = (
    <div
      className='bg-muted text-muted-foreground flex items-center justify-center rounded-full text-xs font-medium'
      style={{ width: size, height: size }}
    >
      {trimmedName.charAt(0).toUpperCase()}
    </div>
  )
  const catalogEntry = iconCatalog.get(baseKey)
  if (!catalogEntry) return fallback

  const hasVariant = segments.length > 1 && /^[A-Z]/.test(segments[1])
  const variant = hasVariant ? segments[1] : ''
  const propStartIndex = hasVariant ? 2 : 1
  const leafVariant =
    variant === 'Color' && catalogEntry.param.hasColor ? 'Color' : 'Mono'
  const isLeaf = !variant || variant === 'Color'
  const iconKey = isLeaf
    ? `${baseKey}${leafVariant === 'Color' ? '.Color' : ''}`
    : `${baseKey}.${variant}`
  let IconComponent = COMMON_ICONS[iconKey as keyof typeof COMMON_ICONS] as
    | ComponentType<Record<string, unknown>>
    | undefined

  if (!IconComponent) {
    IconComponent = lazyIcons.get(iconKey)
    if (!IconComponent) {
      IconComponent = lazy(async () => {
        try {
          if (isLeaf) {
            return await import(
              /* webpackInclude: /[\\/]es[\\/][^\\/]+[\\/]components[\\/](Mono|Color)\.js$/ */
              `@lobehub/icons/es/${baseKey}/components/${leafVariant}.js`
            )
          }
          const module = await import(
            /* webpackInclude: /[\\/]es[\\/][^\\/]+[\\/]index\.js$/ */
            `@lobehub/icons/es/${baseKey}/index.js`
          )
          const icon = module.default as Record<string, unknown>
          const component = icon[variant] ?? module.default
          return { default: component }
        } catch {
          return { default: () => fallback }
        }
      })
      lazyIcons.set(iconKey, IconComponent)
    }
  }

  // Parse chained properties (e.g., "type={'platform'}", "shape='square'")
  const props: Record<string, string | number | boolean> = {}

  for (let i = propStartIndex; i < segments.length; i++) {
    const seg = segments[i]
    if (!seg) continue

    const eqIdx = seg.indexOf('=')
    if (eqIdx === -1) {
      props[seg.trim()] = true
      continue
    }

    const key = seg.slice(0, eqIdx).trim()
    const valRaw = seg.slice(eqIdx + 1).trim()
    props[key] = parseValue(valRaw)
  }

  // Set size if not explicitly specified in the string
  if (props.size == null && size != null) {
    props.size = size
  }

  return (
    <Suspense fallback={fallback}>
      <IconComponent {...props} />
    </Suspense>
  )
}

// The selector uses the same installed icon registry as the renderer.
export function getLobeIconNames(): string[] {
  const names = toc.flatMap((icon) =>
    icon.param.hasColor ? [icon.id, `${icon.id}.Color`] : [icon.id]
  )
  return [...new Set([...names, ...Object.keys(CUSTOM_ICONS)])].sort()
}
