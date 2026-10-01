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
  Globe2,
  ImageIcon,
  Layers3,
  Shield,
  Sparkles,
  Star,
  Zap,
} from 'lucide-react'
import type { ReactNode } from 'react'

import {
  groupTypeCustomIconSchema,
  type GroupTypeIconName,
} from '@/lib/group-types'
import { getLobeIcon, getLobeIconNames } from '@/lib/lobe-icon'

const icons = {
  layers: Layers3,
  sparkles: Sparkles,
  zap: Zap,
  shield: Shield,
  star: Star,
  globe: Globe2,
}

const lobeIconNames = new Set(getLobeIconNames())

export function GroupTypeIcon(props: {
  icon: GroupTypeIconName
  customIcon?: string
  color: string
}) {
  let icon: ReactNode
  if (props.icon === 'custom') {
    const name = props.customIcon ?? ''
    const baseName = name.split('.')[0]
    icon =
      groupTypeCustomIconSchema.safeParse(name).success &&
      lobeIconNames.has(baseName) ? (
        getLobeIcon(name, 16)
      ) : (
        <ImageIcon className='size-4' />
      )
  } else {
    const Icon = icons[props.icon] ?? Layers3
    icon = <Icon className='size-4' />
  }
  return (
    <span
      aria-hidden='true'
      className='inline-flex size-7 shrink-0 items-center justify-center rounded-md'
      style={{ color: props.color, backgroundColor: `${props.color}18` }}
    >
      {icon}
    </span>
  )
}
