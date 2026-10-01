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
import { z } from 'zod'

export type GroupTypeIconName =
  | 'layers'
  | 'sparkles'
  | 'zap'
  | 'shield'
  | 'star'
  | 'globe'

export type GroupType = {
  name: string
  icon: GroupTypeIconName
  color: string
  groups: string[]
}

export const groupTypeIconNames = [
  'layers',
  'sparkles',
  'zap',
  'shield',
  'star',
  'globe',
] as const

export const groupTypeSchema = z.object({
  name: z.string().max(48),
  icon: z.enum(groupTypeIconNames),
  color: z.string().regex(/^#[0-9a-fA-F]{6}$/),
  groups: z.array(z.string().refine((name) => name.trim().length > 0)),
})

export const groupTypesSchema = z
  .array(groupTypeSchema)
  .superRefine((types, ctx) => {
    const names = types.map((type) => type.name.trim())
    const groups = types.flatMap((type) => type.groups)
    if (
      names.some((name) => !name) ||
      new Set(names).size !== names.length ||
      new Set(groups).size !== groups.length
    ) {
      ctx.addIssue({
        code: 'custom',
        message: 'Invalid group type configuration',
      })
    }
  })

export function groupByType<T extends { value: string }>(
  options: T[],
  types: GroupType[]
): { type: GroupType | null; options: T[] }[] {
  const assigned = new Set<string>()
  const sections: { type: GroupType | null; options: T[] }[] = types.map(
    (type) => {
      const names = new Set(type.groups)
      const matching = options.filter(
        (option) => names.has(option.value) && !assigned.has(option.value)
      )
      for (const option of matching) assigned.add(option.value)
      return { type, options: matching }
    }
  )
  const other = options.filter((option) => !assigned.has(option.value))
  if (other.length) sections.push({ type: null, options: other })
  return sections
}
