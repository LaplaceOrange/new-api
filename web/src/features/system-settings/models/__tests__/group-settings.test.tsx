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
import { zodResolver } from '@hookform/resolvers/zod'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { I18nextProvider } from 'react-i18next'
import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'

import { groupTypesSchema } from '@/lib/group-types'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { GroupRatioForm } from '../group-ratio-form'

const defaults = {
  GroupRatio: '{"default":1,"vip":0.8}',
  TopupGroupRatio: '{"vip":1.2}',
  UserUsableGroups: '{"default":"Standard access","vip":"Premium access"}',
  GroupGroupRatio: '{}',
  AutoGroups: '["default","vip"]',
  MaxTokenAutoGroups: 5,
  DefaultUseAutoGroup: false,
  GroupSpecialUsableGroup: '{}',
  GroupTypes: '[]',
}

const schema = z.object({
  GroupRatio: z.string(),
  TopupGroupRatio: z.string(),
  UserUsableGroups: z.string(),
  GroupGroupRatio: z.string(),
  AutoGroups: z.string(),
  MaxTokenAutoGroups: z.number(),
  DefaultUseAutoGroup: z.boolean(),
  GroupSpecialUsableGroup: z.string(),
  GroupTypes: z.string().refine((value) => {
    try {
      return groupTypesSchema.safeParse(JSON.parse(value)).success
    } catch {
      return false
    }
  }, 'Invalid group type configuration'),
})

function Fixture(props: {
  onSave?: (values: typeof defaults) => Promise<void>
  initial?: Partial<typeof defaults>
}) {
  const [actions, setActions] = useState<HTMLDivElement | null>(null)
  const form = useForm({
    defaultValues: { ...defaults, ...props.initial },
    resolver: zodResolver(schema),
  })
  return (
    <SettingsPageProvider actionsContainer={actions}>
      <div ref={setActions} />
      <GroupRatioForm
        form={form}
        onSave={props.onSave ?? (async () => {})}
        isSaving={false}
      />
    </SettingsPageProvider>
  )
}

describe('group type settings', () => {
  it('displays the translated selected icon before opening its menu', async () => {
    const i18n = createInstance()
    await i18n.init({
      lng: 'en',
      resources: { en: { translation: { star: 'Star' } } },
    })
    const user = userEvent.setup()
    render(
      <I18nextProvider i18n={i18n}>
        <Fixture
          initial={{
            GroupTypes:
              '[{"name":"Priority","icon":"star","color":"#059669","groups":["vip"]}]',
          }}
        />
      </I18nextProvider>
    )
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    expect(
      screen.getByRole('combobox', { name: 'Type icon' })
    ).toHaveTextContent('Star')
  })

  it('submits the type name, icon, color and groups without changing ratios', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(<Fixture onSave={onSave} />)
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    expect(screen.getByText('No group types yet.')).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Add type' }))
    await user.type(
      screen.getByRole('textbox', { name: 'Type name' }),
      'Priority'
    )
    fireEvent.change(screen.getByLabelText('Theme color'), {
      target: { value: '#059669' },
    })
    await user.click(screen.getByRole('combobox', { name: 'Type icon' }))
    await user.click(screen.getByRole('option', { name: 'star' }))
    await user.click(
      screen.getByRole('combobox', { name: 'Groups in this type' })
    )
    await user.click(screen.getByRole('option', { name: 'vip' }))
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() => expect(onSave).toHaveBeenCalled())
    const values = onSave.mock.calls[0][0]
    expect(JSON.parse(values.GroupTypes)).toEqual([
      { name: 'Priority', icon: 'star', color: '#059669', groups: ['vip'] },
    ])
    expect(values.GroupRatio).toBe(defaults.GroupRatio)
    expect(values.TopupGroupRatio).toBe(defaults.TopupGroupRatio)
    expect(values.GroupGroupRatio).toBe(defaults.GroupGroupRatio)
  })

  it('reveals invalid types when saving from the ratios tab', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(<Fixture onSave={onSave} />)
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    await user.click(screen.getByRole('button', { name: 'Add type' }))
    await user.click(screen.getByRole('tab', { name: 'Group ratios' }))
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: 'Group types' })).toHaveAttribute(
        'aria-selected',
        'true'
      )
    )
    expect(screen.getByText('Invalid group type configuration')).toBeVisible()
    expect(onSave).not.toHaveBeenCalled()
  })

  it('shows an error instead of crashing for malformed type JSON', async () => {
    const user = userEvent.setup()
    render(<Fixture initial={{ GroupTypes: '[null]' }} />)
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    expect(screen.getByText('Invalid group type configuration')).toBeVisible()
    expect(screen.queryByRole('textbox', { name: 'Type name' })).toBeNull()
  })
})
