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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  Plus,
  Save,
  Trash2,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { getServerErrorMessage } from '@/lib/server-error-message'

import {
  getMonitorGroupModels,
  getMonitorGroups,
  updateDegradationConfig,
} from './api'
import type {
  DegradationConfig,
  DegradationConfigGroup,
  DegradationConfigModel,
} from './types'

function moveItem<T>(items: T[], index: number, direction: -1 | 1): T[] {
  const next = index + direction
  if (next < 0 || next >= items.length) return items
  const copy = [...items]
  const current = copy[index]
  copy[index] = copy[next]
  copy[next] = current
  return copy
}

export function DegradationEditor(props: { config: DegradationConfig }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<DegradationConfig>(props.config)
  const [groupToAdd, setGroupToAdd] = useState('')
  const [modelToAdd, setModelToAdd] = useState<Record<string, string>>({})

  useEffect(() => {
    setDraft(props.config)
  }, [props.config])

  const groupsQuery = useQuery({
    queryKey: ['degradation-groups'],
    queryFn: getMonitorGroups,
  })

  const saveMutation = useMutation({
    mutationFn: updateDegradationConfig,
    onSuccess: async () => {
      toast.success(t('Degradation monitor saved'))
      await queryClient.invalidateQueries({ queryKey: ['degradation'] })
    },
    onError: (error) => {
      toast.error(
        getServerErrorMessage(error, t('Failed to save degradation monitor'))
      )
    },
  })

  const usedGroups = new Set(draft.groups.map((group) => group.group))
  const groupOptions = [
    ...new Set([
      ...(groupsQuery.data?.data ?? []),
      ...draft.groups.map((group) => group.group),
    ]),
  ]

  const updateGroup = (
    index: number,
    update: (group: DegradationConfigGroup) => DegradationConfigGroup
  ) => {
    setDraft((current) => ({
      ...current,
      groups: current.groups.map((group, groupIndex) =>
        groupIndex === index ? update(group) : group
      ),
    }))
  }

  return (
    <Collapsible
      defaultOpen={false}
      className='border-border bg-card overflow-visible rounded-xl border shadow-sm'
    >
      <div className='flex flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-5'>
        <div>
          <h2 className='text-sm font-semibold'>{t('Monitor settings')}</h2>
          <p className='text-muted-foreground mt-1 text-xs'>
            {t('Configure schedules and published models')}
          </p>
        </div>
        <div className='flex items-center gap-3'>
          <div className='flex items-center gap-2'>
            <span className='text-sm'>{t('Enabled')}</span>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(checked) =>
                setDraft((current) => ({ ...current, enabled: checked }))
              }
            />
          </div>
          <CollapsibleTrigger
            className='border-input hover:bg-accent inline-flex h-8 items-center gap-1 rounded-md border px-3 text-sm'
            type='button'
          >
            {t('Edit settings')}
            <ChevronDown className='size-4' />
          </CollapsibleTrigger>
        </div>
      </div>
      <CollapsibleContent className='space-y-5 overflow-visible border-t px-4 py-4 sm:px-5'>
        <div className='grid gap-3 sm:grid-cols-3'>
          <label className='space-y-1 text-sm'>
            <span>{t('Check interval (minutes)')}</span>
            <Input
              type='number'
              min={1}
              value={draft.interval_minutes}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  interval_minutes: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className='space-y-1 text-sm'>
            <span>{t('Retry count')}</span>
            <Input
              type='number'
              min={0}
              value={draft.retry_count}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  retry_count: Number(event.target.value),
                }))
              }
            />
          </label>
          <label className='space-y-1 text-sm'>
            <span>{t('Retry interval (minutes)')}</span>
            <Input
              type='number'
              min={1}
              value={draft.retry_interval_minutes}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  retry_interval_minutes: Number(event.target.value),
                }))
              }
            />
          </label>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          <Select
            value={groupToAdd}
            onValueChange={(value) => setGroupToAdd(value ?? '')}
          >
            <SelectTrigger className='w-56'>
              <SelectValue placeholder={t('Select group')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              {groupOptions.length > 0 ? (
                groupOptions.map((group) => (
                  <SelectItem
                    key={group}
                    value={group}
                    disabled={usedGroups.has(group)}
                  >
                    {group}
                  </SelectItem>
                ))
              ) : (
                <SelectItem value='__no-groups__' disabled>
                  {t('No groups available')}
                </SelectItem>
              )}
            </SelectContent>
          </Select>
          <Button
            type='button'
            variant='outline'
            disabled={!groupToAdd}
            onClick={() => {
              setDraft((current) => ({
                ...current,
                groups: [
                  ...current.groups,
                  {
                    group: groupToAdd,
                    sort: current.groups.length,
                    models: [],
                  },
                ],
              }))
              setGroupToAdd('')
            }}
          >
            <Plus />
            {t('Add group')}
          </Button>
        </div>
        <div className='space-y-6'>
          {draft.groups.map((group, index) => (
            <GroupEditor
              key={group.group}
              group={group}
              modelToAdd={modelToAdd[group.group] ?? ''}
              onModelToAdd={(value) =>
                setModelToAdd((current) => ({
                  ...current,
                  [group.group]: value,
                }))
              }
              onChange={(next) => updateGroup(index, () => next)}
              onMove={(direction) =>
                setDraft((current) => ({
                  ...current,
                  groups: moveItem(current.groups, index, direction),
                }))
              }
              onRemove={() =>
                setDraft((current) => ({
                  ...current,
                  groups: current.groups.filter(
                    (item) => item.group !== group.group
                  ),
                }))
              }
              disableUp={index === 0}
              disableDown={index === draft.groups.length - 1}
            />
          ))}
        </div>
        <Button
          type='button'
          disabled={saveMutation.isPending}
          onClick={() => saveMutation.mutate(draft)}
        >
          <Save /> {t('Save')}
        </Button>
      </CollapsibleContent>
    </Collapsible>
  )
}

function GroupEditor(props: {
  group: DegradationConfigGroup
  modelToAdd: string
  onModelToAdd: (value: string) => void
  onChange: (group: DegradationConfigGroup) => void
  onMove: (direction: -1 | 1) => void
  onRemove: () => void
  disableUp: boolean
  disableDown: boolean
}) {
  const { t } = useTranslation()
  const modelsQuery = useQuery({
    queryKey: ['degradation-models', props.group.group],
    queryFn: () => getMonitorGroupModels(props.group.group),
  })
  const used = new Set(props.group.models.map((item) => item.model))
  const modelOptions = [
    ...new Set([
      ...(modelsQuery.data?.data ?? []),
      ...props.group.models.map((model) => model.model),
    ]),
  ]

  const updateModel = (
    index: number,
    update: (model: DegradationConfigModel) => DegradationConfigModel
  ) => {
    props.onChange({
      ...props.group,
      models: props.group.models.map((model, modelIndex) =>
        modelIndex === index ? update(model) : model
      ),
    })
  }

  return (
    <div className='border-border bg-card space-y-4 rounded-xl border p-4 shadow-sm sm:p-5'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 className='text-sm font-medium'>{props.group.group}</h3>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          disabled={props.disableUp}
          onClick={() => props.onMove(-1)}
          title={t('Move up')}
        >
          <ArrowUp />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          disabled={props.disableDown}
          onClick={() => props.onMove(1)}
          title={t('Move down')}
        >
          <ArrowDown />
        </Button>
        <Button
          type='button'
          variant='ghost'
          size='icon'
          onClick={props.onRemove}
          title={t('Remove')}
        >
          <Trash2 />
        </Button>
      </div>
      <div className='flex flex-wrap items-center gap-2'>
        <Select
          value={props.modelToAdd}
          onValueChange={(value) => props.onModelToAdd(value ?? '')}
        >
          <SelectTrigger className='w-64'>
            <SelectValue placeholder={t('Select model')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            {modelOptions.length > 0 ? (
              modelOptions.map((model) => (
                <SelectItem
                  key={model}
                  value={model}
                  disabled={used.has(model)}
                >
                  {model}
                </SelectItem>
              ))
            ) : (
              <SelectItem value='__no-models__' disabled>
                {t('No models available')}
              </SelectItem>
            )}
          </SelectContent>
        </Select>
        <Button
          type='button'
          variant='outline'
          disabled={!props.modelToAdd}
          onClick={() => {
            props.onChange({
              ...props.group,
              models: [
                ...props.group.models,
                {
                  model: props.modelToAdd,
                  expected: '',
                  sort: props.group.models.length,
                },
              ],
            })
            props.onModelToAdd('')
          }}
        >
          <Plus />
          {t('Add model')}
        </Button>
      </div>
      <div className='border-border divide-border divide-y overflow-visible rounded-lg border'>
        {props.group.models.map((model, index) => (
          <div
            key={model.model}
            className='grid gap-2 py-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] md:items-center'
          >
            <div className='truncate text-sm'>{model.model}</div>
            <Input
              value={model.expected}
              placeholder={t('Expected detected name')}
              onChange={(event) =>
                updateModel(index, (current) => ({
                  ...current,
                  expected: event.target.value,
                }))
              }
            />
            <div className='flex items-center gap-1'>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                disabled={index === 0}
                onClick={() =>
                  props.onChange({
                    ...props.group,
                    models: moveItem(props.group.models, index, -1),
                  })
                }
                title={t('Move up')}
              >
                <ArrowUp />
              </Button>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                disabled={index === props.group.models.length - 1}
                onClick={() =>
                  props.onChange({
                    ...props.group,
                    models: moveItem(props.group.models, index, 1),
                  })
                }
                title={t('Move down')}
              >
                <ArrowDown />
              </Button>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                onClick={() =>
                  props.onChange({
                    ...props.group,
                    models: props.group.models.filter(
                      (item) => item.model !== model.model
                    ),
                  })
                }
                title={t('Remove')}
              >
                <Trash2 />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
