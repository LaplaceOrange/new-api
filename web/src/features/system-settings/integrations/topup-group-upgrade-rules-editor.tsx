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
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'

import { safeJsonParseWithValidation } from '../utils/json-parser'
import { isArray, isObjectRecord } from '../utils/json-validators'

export type TopUpGroupUpgradeRule = {
  type: 'single' | 'cumulative'
  amount: string
  group: string
}

type TopUpGroupUpgradeRulesEditorProps = {
  value: string
  groupRatio: string
  onChange: (value: string) => void
}

function parseRules(value: string): TopUpGroupUpgradeRule[] {
  const parsed = safeJsonParseWithValidation<unknown[]>(value, {
    fallback: [],
    validator: isArray,
    silent: true,
  })

  return parsed.flatMap((item) => {
    if (!isObjectRecord(item)) return []
    if (
      (item.type !== 'single' && item.type !== 'cumulative') ||
      typeof item.amount !== 'string' ||
      typeof item.group !== 'string'
    ) {
      return []
    }
    return [
      {
        type: item.type,
        amount: item.amount,
        group: item.group,
      },
    ]
  })
}

function parseGroupNames(groupRatio: string): string[] {
  const parsed = safeJsonParseWithValidation<Record<string, unknown>>(
    groupRatio,
    {
      fallback: {},
      validator: isObjectRecord,
      silent: true,
    }
  )
  return Object.keys(parsed).sort((left, right) => left.localeCompare(right))
}

export function TopUpGroupUpgradeRulesEditor({
  value,
  groupRatio,
  onChange,
}: TopUpGroupUpgradeRulesEditorProps) {
  const { t } = useTranslation()
  const rules = useMemo(() => parseRules(value), [value])
  const groupOptions = useMemo(() => parseGroupNames(groupRatio), [groupRatio])

  const updateRules = (nextRules: TopUpGroupUpgradeRule[]) => {
    onChange(JSON.stringify(nextRules, null, 2))
  }

  const updateRule = (index: number, patch: Partial<TopUpGroupUpgradeRule>) => {
    updateRules(
      rules.map((rule, ruleIndex) =>
        ruleIndex === index ? { ...rule, ...patch } : rule
      )
    )
  }

  const moveRule = (index: number, direction: 'up' | 'down') => {
    const nextIndex = direction === 'up' ? index - 1 : index + 1
    if (nextIndex < 0 || nextIndex >= rules.length) return

    const nextRules = [...rules]
    ;[nextRules[index], nextRules[nextIndex]] = [
      nextRules[nextIndex],
      nextRules[index],
    ]
    updateRules(nextRules)
  }

  return (
    <section
      aria-label={t('Top-up group upgrades')}
      className='space-y-4'
      data-slot='topup-group-upgrade-rules'
    >
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div className='space-y-1'>
          <p className='text-muted-foreground text-sm'>
            {t(
              "Automatically switch a user's group when a completed top-up reaches a configured threshold."
            )}
          </p>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Rules are checked from highest to lowest priority. The first matching rule is applied.'
            )}
          </p>
        </div>
        <Button
          type='button'
          size='sm'
          className='w-full sm:w-auto'
          disabled={groupOptions.length === 0}
          onClick={() =>
            updateRules([
              ...rules,
              {
                type: 'single',
                amount: '',
                group: groupOptions[0] ?? '',
              },
            ])
          }
        >
          <Plus className='mr-2 h-4 w-4' />
          {t('Add upgrade rule')}
        </Button>
      </div>

      {groupOptions.length === 0 && (
        <p className='text-muted-foreground rounded-md border border-dashed p-3 text-sm'>
          {t('Create a group before adding upgrade rules.')}
        </p>
      )}

      {rules.length === 0 ? (
        <div className='text-muted-foreground rounded-lg border border-dashed p-6 text-center text-sm'>
          {t('No upgrade rules configured.')}
        </div>
      ) : (
        <div className='space-y-3'>
          {rules.map((rule, index) => {
            const priority = index + 1
            const duplicateCount = rules
              .slice(0, index)
              .filter(
                (previousRule) =>
                  previousRule.type === rule.type &&
                  previousRule.amount === rule.amount &&
                  previousRule.group === rule.group
              ).length
            const ruleKey = `${rule.type}:${rule.amount}:${rule.group}:${duplicateCount}`
            const selectableGroups = groupOptions.includes(rule.group)
              ? groupOptions
              : [rule.group, ...groupOptions]
            const amountId = `topup-group-upgrade-rule-${index}-amount`
            const typeId = `topup-group-upgrade-rule-${index}-type`
            const groupId = `topup-group-upgrade-rule-${index}-group`

            return (
              <div
                key={ruleKey}
                className='rounded-lg border p-4'
                data-slot='topup-group-upgrade-rule'
              >
                <div className='mb-4 flex items-center justify-between gap-3'>
                  <span className='text-sm font-medium'>
                    {t('Priority {{priority}}', { priority })}
                  </span>
                  <div className='flex items-center gap-1'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon-sm'
                      disabled={index === 0}
                      aria-label={t('Move upgrade rule {{priority}} up', {
                        priority,
                      })}
                      onClick={() => moveRule(index, 'up')}
                    >
                      <ArrowUp className='h-4 w-4' />
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon-sm'
                      disabled={index === rules.length - 1}
                      aria-label={t('Move upgrade rule {{priority}} down', {
                        priority,
                      })}
                      onClick={() => moveRule(index, 'down')}
                    >
                      <ArrowDown className='h-4 w-4' />
                    </Button>
                    <Button
                      type='button'
                      variant='ghost'
                      size='icon-sm'
                      aria-label={t('Remove upgrade rule {{priority}}', {
                        priority,
                      })}
                      onClick={() =>
                        updateRules(
                          rules.filter((_, ruleIndex) => ruleIndex !== index)
                        )
                      }
                    >
                      <Trash2 className='h-4 w-4' />
                    </Button>
                  </div>
                </div>

                <div className='grid gap-4 md:grid-cols-3'>
                  <div className='space-y-2'>
                    <Label htmlFor={typeId}>{t('Condition')}</Label>
                    <NativeSelect
                      id={typeId}
                      className='w-full'
                      value={rule.type}
                      onChange={(event) =>
                        updateRule(index, {
                          type: event.target
                            .value as TopUpGroupUpgradeRule['type'],
                        })
                      }
                    >
                      <NativeSelectOption value='single'>
                        {t('Single top-up')}
                      </NativeSelectOption>
                      <NativeSelectOption value='cumulative'>
                        {t('Cumulative top-ups')}
                      </NativeSelectOption>
                    </NativeSelect>
                  </div>

                  <div className='space-y-2'>
                    <Label htmlFor={amountId}>{t('Charged amount')}</Label>
                    <Input
                      id={amountId}
                      type='number'
                      min='0.01'
                      step='0.01'
                      inputMode='decimal'
                      value={rule.amount}
                      onChange={(event) =>
                        updateRule(index, { amount: event.target.value })
                      }
                    />
                  </div>

                  <div className='space-y-2'>
                    <Label htmlFor={groupId}>{t('Target group')}</Label>
                    <NativeSelect
                      id={groupId}
                      className='w-full'
                      value={rule.group}
                      onChange={(event) =>
                        updateRule(index, { group: event.target.value })
                      }
                    >
                      <NativeSelectOption value='' disabled>
                        {t('Select a group')}
                      </NativeSelectOption>
                      {selectableGroups.map((group) => (
                        <NativeSelectOption
                          key={group}
                          value={group}
                          disabled={!groupOptions.includes(group)}
                        >
                          {groupOptions.includes(group)
                            ? group
                            : t('Unavailable group: {{group}}', { group })}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}
