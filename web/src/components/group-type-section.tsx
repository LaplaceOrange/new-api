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
import { ChevronDown } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import type { GroupType } from '@/lib/group-types'

import { GroupTypeIcon } from './group-type-icon'
import { Button } from './ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from './ui/collapsible'

type GroupTypeSectionProps = {
  type: GroupType | null
  count: number
  children: ReactNode
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
}

export function GroupTypeSection(props: GroupTypeSectionProps) {
  const { t } = useTranslation()
  const name = props.type?.name ?? t('Other groups')
  return (
    <Collapsible
      open={props.open}
      defaultOpen={props.defaultOpen ?? false}
      onOpenChange={props.onOpenChange}
    >
      <CollapsibleTrigger
        aria-label={name}
        title={name}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.stopPropagation()
          }
        }}
        render={
          <Button
            type='button'
            variant='ghost'
            className='group/type h-10 w-full min-w-0 justify-start gap-2 rounded-md px-2 text-left'
          />
        }
      >
        <GroupTypeIcon
          icon={props.type?.icon ?? 'layers'}
          color={props.type?.color ?? '#737373'}
        />
        <span className='min-w-0 flex-1 truncate text-sm font-medium'>
          {name}
        </span>
        <span
          aria-hidden='true'
          className='text-muted-foreground text-xs tabular-nums'
        >
          {props.count}
        </span>
        <ChevronDown
          aria-hidden='true'
          className='text-muted-foreground size-4 transition-transform group-aria-expanded/type:rotate-180'
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        {props.count ? (
          props.children
        ) : (
          <p className='text-muted-foreground px-3 py-2 text-xs'>
            {t('No available groups')}
          </p>
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}
