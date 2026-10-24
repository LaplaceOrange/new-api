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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { I18nextProvider } from 'react-i18next'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { z } from 'zod'

import { GroupTypeSection } from '@/components/group-type-section'
import { groupTypesSchema } from '@/lib/group-types'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { GroupRatioForm } from '../group-ratio-form'

const { getGroupModelCandidatesMock } = vi.hoisted(() => ({
  getGroupModelCandidatesMock: vi.fn(),
}))

vi.mock('@/features/system-settings/api', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@/features/system-settings/api')>()
  return { ...actual, getGroupModelCandidates: getGroupModelCandidatesMock }
})

const defaults = {
  GroupRatio: '{"default":1,"vip":0.8}',
  GroupModelPolicies: '{}',
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
  GroupModelPolicies: z.string(),
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
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { retry: false },
          mutations: { retry: false },
        },
      })
  )
  const form = useForm({
    defaultValues: { ...defaults, ...props.initial },
    resolver: zodResolver(schema),
  })
  return (
    <QueryClientProvider client={queryClient}>
      <SettingsPageProvider actionsContainer={actions}>
        <div ref={setActions} />
        <GroupRatioForm
          form={form}
          onSave={props.onSave ?? (async () => {})}
          isSaving={false}
        />
      </SettingsPageProvider>
    </QueryClientProvider>
  )
}

describe('group type settings', () => {
  beforeEach(() => {
    getGroupModelCandidatesMock.mockReset()
    getGroupModelCandidatesMock.mockResolvedValue({
      success: true,
      data: ['blocked-model', 'allowed-model'],
    })
  })

  it('saves a custom LobeHub icon ID selected from suggestions', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(
      <Fixture
        onSave={onSave}
        initial={{
          GroupTypes:
            '[{"name":"Priority","icon":"star","color":"#059669","groups":["vip"]}]',
        }}
      />
    )
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    await user.click(screen.getByRole('combobox', { name: 'Type icon' }))
    await user.click(screen.getByRole('option', { name: 'Custom' }))
    await user.click(screen.getByRole('combobox', { name: 'LobeHub icon ID' }))
    await user.paste('DeepSeek.Color')
    await user.click(screen.getByRole('option', { name: 'DeepSeek.Color' }))
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(JSON.parse(onSave.mock.calls[0][0].GroupTypes)).toEqual([
      {
        name: 'Priority',
        icon: 'custom',
        custom_icon: 'DeepSeek.Color',
        color: '#059669',
        groups: ['vip'],
      },
    ])
  })

  it('removes the custom icon ID when saving a preset', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(
      <Fixture
        onSave={onSave}
        initial={{
          GroupTypes:
            '[{"name":"Priority","icon":"custom","custom_icon":"DeepSeek.Color","color":"#059669","groups":["vip"]}]',
        }}
      />
    )
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    await user.click(screen.getByRole('combobox', { name: 'Type icon' }))
    await user.click(screen.getByRole('option', { name: 'shield' }))
    expect(
      screen.queryByRole('combobox', { name: 'LobeHub icon ID' })
    ).toBeNull()
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(JSON.parse(onSave.mock.calls[0][0].GroupTypes)[0]).toEqual({
      name: 'Priority',
      icon: 'shield',
      color: '#059669',
      groups: ['vip'],
    })
  })

  it('restores a saved custom icon ID and keeps its type header collapsed', async () => {
    const user = userEvent.setup()
    const type = {
      name: 'Provider',
      icon: 'custom' as const,
      custom_icon: 'DeepSeek.Color',
      color: '#059669',
      groups: [],
    }
    render(<Fixture initial={{ GroupTypes: JSON.stringify([type]) }} />)
    await user.click(screen.getByRole('tab', { name: 'Group types' }))
    expect(
      screen.getByRole('combobox', { name: 'LobeHub icon ID' })
    ).toHaveValue('DeepSeek.Color')
    render(
      <GroupTypeSection type={type} count={0}>
        {null}
      </GroupTypeSection>
    )
    const header = screen.getByRole('button', { name: 'Provider' })
    expect(header).toHaveAttribute('aria-expanded', 'false')
    expect(header).toBeVisible()
  })

  it('rejects missing IDs and icon property expressions when saving custom icons', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(
      <Fixture
        onSave={onSave}
        initial={{
          GroupTypes:
            '[{"name":"Provider","icon":"custom","color":"#059669","groups":[]}]',
        }}
      />
    )
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() =>
      expect(
        screen.getAllByText('Invalid group type configuration')[0]
      ).toBeVisible()
    )
    expect(onSave).not.toHaveBeenCalled()
    expect(
      groupTypesSchema.safeParse([
        {
          name: 'Provider',
          icon: 'custom',
          custom_icon: 'OpenAI.size={999}',
          color: '#059669',
          groups: [],
        },
      ]).success
    ).toBe(false)
  })

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

  it('keeps blacklist and whitelist selections when switching modes', async () => {
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(<Fixture onSave={onSave} />)

    await user.click(
      within(screen.getByRole('row', { name: /default/ })).getByRole('button', {
        name: 'Details',
      })
    )
    await screen.findByRole('heading', { name: 'Model access policy' })

    const mode = () =>
      screen.getByRole('combobox', { name: 'Model access mode' })
    await user.click(mode())
    await user.click(
      await screen.findByRole('option', { name: 'Disable blacklisted models' })
    )
    await user.click(
      await screen.findByRole('combobox', { name: 'Blacklisted models' })
    )
    await user.click(
      await screen.findByRole('option', { name: 'blocked-model' })
    )
    await user.keyboard('{Escape}')

    await user.click(mode())
    await user.click(
      await screen.findByRole('option', {
        name: 'Allow only whitelisted models',
      })
    )
    await user.click(
      await screen.findByRole('combobox', { name: 'Whitelisted models' })
    )
    await user.click(
      await screen.findByRole('option', { name: 'allowed-model' })
    )
    await user.keyboard('{Escape}')

    await user.click(mode())
    await user.click(
      await screen.findByRole('option', { name: 'Disable blacklisted models' })
    )
    expect(screen.getByText('blocked-model')).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'Close' }))
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(
      JSON.parse(onSave.mock.calls[0][0].GroupModelPolicies).default
    ).toEqual({
      mode: 'blacklist',
      blacklist: ['blocked-model'],
      whitelist: ['allowed-model'],
    })
  })

  it('keeps saved models outside the current candidates removable', async () => {
    getGroupModelCandidatesMock.mockResolvedValue({
      success: true,
      data: ['available-model'],
    })
    const user = userEvent.setup()
    const onSave = vi.fn(async (_values: typeof defaults) => {})
    render(
      <Fixture
        onSave={onSave}
        initial={{
          GroupModelPolicies: JSON.stringify({
            default: {
              mode: 'blacklist',
              blacklist: ['retired-model'],
              whitelist: [],
            },
          }),
        }}
      />
    )

    await user.click(
      within(screen.getByRole('row', { name: /default/ })).getByRole('button', {
        name: 'Details',
      })
    )
    await screen.findByRole('heading', { name: 'Model access policy' })
    expect(screen.getByText('retired-model')).toBeVisible()
    await user.click(
      screen.getByRole('button', { name: 'Remove retired-model' })
    )
    await user.click(screen.getByRole('button', { name: 'Close' }))
    await user.click(screen.getByRole('button', { name: 'Save group ratios' }))
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1))
    expect(
      JSON.parse(onSave.mock.calls[0][0].GroupModelPolicies).default.blacklist
    ).toEqual([])
  })

  it('loads only the selected group candidates and refreshes them when opening another group', async () => {
    getGroupModelCandidatesMock.mockImplementation(async (group: string) => ({
      success: true,
      data: [`${group}-model`],
    }))
    const user = userEvent.setup()
    render(
      <Fixture
        initial={{
          GroupModelPolicies: JSON.stringify({
            default: { mode: 'blacklist', blacklist: [], whitelist: [] },
            vip: { mode: 'whitelist', blacklist: [], whitelist: [] },
          }),
        }}
      />
    )

    await user.click(
      within(screen.getByRole('row', { name: /default/ })).getByRole('button', {
        name: 'Details',
      })
    )
    await user.click(
      await screen.findByRole('combobox', { name: 'Blacklisted models' })
    )
    expect(
      await screen.findByRole('option', { name: 'default-model' })
    ).toBeVisible()
    expect(screen.queryByRole('option', { name: 'vip-model' })).toBeNull()
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('button', { name: 'Close' }))

    await user.click(
      within(screen.getByRole('row', { name: /vip/ })).getByRole('button', {
        name: 'Details',
      })
    )
    await user.click(
      await screen.findByRole('combobox', { name: 'Whitelisted models' })
    )
    expect(
      await screen.findByRole('option', { name: 'vip-model' })
    ).toBeVisible()
    expect(screen.queryByRole('option', { name: 'default-model' })).toBeNull()
  })

  it('keeps saved selections editable after a candidate load error and offers retry', async () => {
    getGroupModelCandidatesMock.mockRejectedValueOnce(new Error('offline'))
    const user = userEvent.setup()
    render(
      <Fixture
        initial={{
          GroupModelPolicies: JSON.stringify({
            default: {
              mode: 'whitelist',
              blacklist: [],
              whitelist: ['saved-model'],
            },
          }),
        }}
      />
    )
    await user.click(
      within(screen.getByRole('row', { name: /default/ })).getByRole('button', {
        name: 'Details',
      })
    )
    expect(await screen.findByText('Failed to load models')).toBeVisible()
    expect(
      screen.getByRole('button', { name: 'Remove saved-model' })
    ).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() =>
      expect(screen.queryByText('Failed to load models')).toBeNull()
    )
  })
})
