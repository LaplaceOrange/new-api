import { z } from 'zod'

import type { Upstream, UpstreamPayload } from '../types'

export const upstreamEditorSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, 'Name is required')
      .max(128, 'Name is too long'),
    addresses: z.string().trim().min(1, 'Enter at least one upstream address'),
    primary_url: z.string().trim().min(1, 'Select a primary address'),
    user_agent: z.string().trim().max(512),
    auto_refresh_token: z.boolean(),
    access_token: z.string(),
    refresh_token: z.string(),
  })
  .superRefine((value, ctx) => {
    const addresses = value.addresses
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean)
    if (addresses.length > 32) {
      ctx.addIssue({
        code: 'custom',
        path: ['addresses'],
        message: 'Use at most 32 upstream addresses',
      })
    }
    if (new Set(addresses).size !== addresses.length) {
      ctx.addIssue({
        code: 'custom',
        path: ['addresses'],
        message: 'Upstream addresses must be unique',
      })
    }
    if (!addresses.includes(value.primary_url)) {
      ctx.addIssue({
        code: 'custom',
        path: ['primary_url'],
        message: 'Select a primary address from the address list',
      })
    }
    for (const address of addresses) {
      try {
        const url = new URL(address)
        if (
          !['https:', 'http:'].includes(url.protocol) ||
          url.username ||
          url.password ||
          url.search ||
          url.hash
        ) {
          throw new Error()
        }
      } catch {
        ctx.addIssue({
          code: 'custom',
          path: ['addresses'],
          message:
            'Enter valid HTTP or HTTPS addresses without credentials, query strings, or fragments',
        })
        break
      }
    }
  })

export type UpstreamEditorValues = z.infer<typeof upstreamEditorSchema>

export function upstreamEditorDefaults(
  upstream: Upstream | null
): UpstreamEditorValues {
  return {
    name: upstream?.name ?? '',
    addresses: upstream?.addresses.join('\n') ?? '',
    primary_url: upstream?.primary_url ?? '',
    user_agent: upstream?.user_agent ?? '',
    auto_refresh_token: upstream?.auto_refresh_token ?? false,
    access_token: '',
    refresh_token: '',
  }
}

export function upstreamEditorPayload(
  values: UpstreamEditorValues
): UpstreamPayload {
  const payload: UpstreamPayload = {
    name: values.name.trim(),
    primary_url: values.primary_url.trim(),
    addresses: values.addresses
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean),
    user_agent: values.user_agent.trim(),
    auto_refresh_token: values.auto_refresh_token,
  }
  if (values.access_token.trim()) {
    payload.access_token = values.access_token.trim()
  }
  if (values.refresh_token.trim()) {
    payload.refresh_token = values.refresh_token.trim()
  }
  return payload
}

export async function upstreamRequestHash(serialized: string): Promise<string> {
  const hash = await crypto.subtle.digest(
    'SHA-256',
    new TextEncoder().encode(serialized)
  )
  return Array.from(new Uint8Array(hash), (byte) =>
    byte.toString(16).padStart(2, '0')
  ).join('')
}
