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
import { Plus, Trash2 } from 'lucide-react'
import { useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { GroupTypeIcon } from '@/components/group-type-icon'
import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  groupTypeIconNames,
  groupTypeSchema,
  type GroupType,
  type GroupTypeIconName,
} from '@/lib/group-types'

type GroupTypeEditorProps = {
  value: string
  groups: string[]
  onChange: (value: string) => void
}

export function GroupTypeEditor(props: GroupTypeEditorProps) {
  const { t } = useTranslation()
  const keys = useRef<string[]>([])
  const types = useMemo(() => {
    try {
      const parsed: unknown = JSON.parse(props.value)
      const result = z.array(groupTypeSchema).safeParse(parsed)
      return result.success ? result.data : null
    } catch {
      return null
    }
  }, [props.value])
  if (types === null) {
    return (
      <ErrorState
        title={t('Invalid group type configuration')}
        className='min-h-32'
      />
    )
  }
  while (keys.current.length < types.length) {
    keys.current.push(crypto.randomUUID())
  }

  const changeType = (index: number, next: GroupType) => {
    props.onChange(
      JSON.stringify(
        types.map((type, position) => (position === index ? next : type)),
        null,
        2
      )
    )
  }

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-3'>
        <div className='min-w-0'>
          <h3 className='text-sm font-semibold'>{t('Group types')}</h3>
        </div>
        <Button
          type='button'
          size='sm'
          onClick={() => {
            keys.current.push(crypto.randomUUID())
            props.onChange(
              JSON.stringify(
                [
                  ...types,
                  {
                    name: '',
                    icon: 'layers',
                    color: '#2563eb',
                    groups: [],
                  },
                ],
                null,
                2
              )
            )
          }}
        >
          <Plus className='size-4' />
          {t('Add type')}
        </Button>
      </div>
      {types.length === 0 && (
        <EmptyState title={t('No group types yet.')} className='min-h-32' />
      )}
      {types.map((type, index) => {
        const assignedElsewhere = new Set(
          types.flatMap((entry, position) =>
            position === index ? [] : entry.groups
          )
        )
        return (
          <div
            className='grid gap-3 border-b pb-4 last:border-b-0'
            key={keys.current[index]}
          >
            <div className='flex items-center gap-2'>
              <GroupTypeIcon icon={type.icon} color={type.color} />
              <Input
                aria-label={t('Type name')}
                className='min-w-0 flex-1'
                maxLength={48}
                placeholder={t('Type name')}
                value={type.name}
                onChange={(event) =>
                  changeType(index, { ...type, name: event.target.value })
                }
              />
              <Button
                type='button'
                variant='ghost'
                size='icon'
                aria-label={t('Remove type {{name}}', {
                  name: type.name || index + 1,
                })}
                onClick={() => {
                  keys.current.splice(index, 1)
                  props.onChange(
                    JSON.stringify(
                      types.filter((_, position) => position !== index),
                      null,
                      2
                    )
                  )
                }}
              >
                <Trash2 className='size-4' />
              </Button>
            </div>
            <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_8rem]'>
              <div className='space-y-1'>
                <Label htmlFor={`group-type-icon-${keys.current[index]}`}>
                  {t('Type icon')}
                </Label>
                <Select
                  value={type.icon}
                  onValueChange={(icon) =>
                    changeType(index, {
                      ...type,
                      icon: icon as GroupTypeIconName,
                    })
                  }
                >
                  <SelectTrigger
                    id={`group-type-icon-${keys.current[index]}`}
                    className='w-full'
                    aria-label={t('Type icon')}
                  >
                    <SelectValue>
                      <GroupTypeIcon icon={type.icon} color={type.color} />
                      {t(type.icon)}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {groupTypeIconNames.map((icon) => (
                        <SelectItem key={icon} value={icon}>
                          <GroupTypeIcon icon={icon} color={type.color} />
                          {t(icon)}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <div className='space-y-1'>
                <Label htmlFor={`group-type-color-${keys.current[index]}`}>
                  {t('Theme color')}
                </Label>
                <Input
                  id={`group-type-color-${keys.current[index]}`}
                  type='color'
                  className='w-full cursor-pointer p-1'
                  value={type.color}
                  onChange={(event) =>
                    changeType(index, { ...type, color: event.target.value })
                  }
                />
              </div>
            </div>
            <div className='space-y-1'>
              <Label htmlFor={`group-type-groups-${keys.current[index]}`}>
                {t('Groups in this type')}
              </Label>
              <MultiSelect
                id={`group-type-groups-${keys.current[index]}`}
                aria-label={t('Groups in this type')}
                options={props.groups
                  .filter((group) => !assignedElsewhere.has(group))
                  .map((group) => ({ label: group, value: group }))}
                selected={type.groups}
                onChange={(groups) => changeType(index, { ...type, groups })}
                placeholder={t('Select groups')}
              />
            </div>
          </div>
        )
      })}
    </div>
  )
}
