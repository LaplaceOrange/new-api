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
  fireEvent,
  render,
  screen,
  within,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, test } from 'vitest'

let shouldReduceMotion = false
const reducedMotionMediaQuery = window.matchMedia('(prefers-reduced-motion)')
Object.defineProperty(reducedMotionMediaQuery, 'matches', {
  configurable: true,
  get: () => shouldReduceMotion,
})
Object.defineProperty(window, 'matchMedia', {
  configurable: true,
  value: () => reducedMotionMediaQuery,
})

function setReducedMotion(value: boolean) {
  shouldReduceMotion = value
  reducedMotionMediaQuery.dispatchEvent(new Event('change'))
}

const { useState } = await import('react')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ApiKeyGroupCombobox } = await import('../api-key-group-combobox')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Auto: 'Auto',
        Ratio: 'Ratio',
        'Search...': 'Search...',
        'No group found.': 'No group found.',
        'Select a group': 'Select a group',
        'Top-up ratio': 'Top-up ratio',
        'Effective ratio': 'Effective ratio',
      },
    },
  },
})

const options = [
  {
    value: 'auto',
    label: 'auto',
    desc: 'Global automatic routing',
    ratio: '自动',
  },
  { value: 'default', label: 'default', desc: 'User group', ratio: 1 },
  {
    value: 'vip',
    label: 'vip',
    desc: 'Priority group',
    ratio: 3,
    topupRatio: 0.1,
  },
]

function Harness(props: {
  initialValue: string
  options?: typeof options
  groupTypes?: React.ComponentProps<typeof ApiKeyGroupCombobox>['groupTypes']
}) {
  const [value, setValue] = useState(props.initialValue)

  return (
    <I18nextProvider i18n={i18n}>
      <ApiKeyGroupCombobox
        options={props.options ?? options}
        groupTypes={props.groupTypes}
        value={value}
        onValueChange={setValue}
      />
      <output data-testid='selected-group'>{value}</output>
    </I18nextProvider>
  )
}

function getTrigger(): HTMLButtonElement {
  return screen.getByRole('combobox')
}

function getCommandItem(label: string): HTMLElement {
  const item = [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].find((candidate) => candidate.textContent?.includes(label))
  if (!item) {
    throw new Error(`Expected command item containing "${label}"`)
  }
  return item
}

describe('API key group combobox Auto effect', () => {
  test('shows the original multiplier struck through before a personal override', () => {
    render(
      <Harness
        initialValue='vip'
        options={options.map((option) =>
          option.value === 'vip' ? { ...option, baseRatio: 2 } : option
        )}
      />
    )
    const trigger = getTrigger()
    expect(within(trigger).getByText('2x', { selector: 'del' })).toBeVisible()
    expect(within(trigger).getByText('3x')).toBeVisible()
    fireEvent.click(trigger)
    expect(
      within(getCommandItem('vip')).getByText('2x', { selector: 'del' })
    ).toBeInTheDocument()
  })

  test('shows type headers and expands an assigned group without changing the selected key group', async () => {
    const user = userEvent.setup()
    render(
      <Harness
        initialValue='auto'
        groupTypes={[
          {
            name: 'Priority tier',
            icon: 'star',
            color: '#059669',
            groups: ['vip'],
          },
          {
            name: 'Standard tier',
            icon: 'layers',
            color: '#2563eb',
            groups: ['default'],
          },
          {
            name: 'Empty tier',
            icon: 'shield',
            color: '#2563eb',
            groups: [],
          },
        ]}
      />
    )
    await user.click(getTrigger())
    const priority = screen.getByRole('button', { name: /Priority tier/ })
    expect(priority).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Priority group')).toBeNull()
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /Standard tier/ })
      ).toBeVisible()
    )
    expect(screen.getByRole('button', { name: /Other groups/ })).toBeVisible()

    priority.focus()
    await user.keyboard('{Enter}')
    expect(priority).toHaveAttribute('aria-expanded', 'true')
    await user.click(getCommandItem('Priority group'))
    expect(screen.getByTestId('selected-group')).toHaveTextContent('vip')
    expect(getTrigger()).toHaveAttribute('aria-expanded', 'false')
  })

  test('search expands matching type groups and allows selecting a hidden result', async () => {
    const user = userEvent.setup()
    render(
      <Harness
        initialValue='auto'
        groupTypes={[
          {
            name: 'Priority tier',
            icon: 'star',
            color: '#059669',
            groups: ['vip'],
          },
          {
            name: 'Empty tier',
            icon: 'shield',
            color: '#2563eb',
            groups: [],
          },
        ]}
      />
    )
    await user.click(getTrigger())
    fireEvent.input(screen.getByPlaceholderText('Search...'), {
      target: { value: 'priority tier' },
    })
    await waitFor(() => expect(getCommandItem('Priority group')).toBeVisible())
    expect(screen.getByRole('button', { name: 'Empty tier' })).toHaveAttribute(
      'aria-expanded',
      'false'
    )
    expect(screen.queryByText('User group')).toBeNull()
    await user.click(getCommandItem('Priority group'))
    expect(screen.getByTestId('selected-group')).toHaveTextContent('vip')
  })

  test('keeps every type visible when searching finds no groups and restores focus on escape', async () => {
    const user = userEvent.setup()
    render(
      <Harness
        initialValue='auto'
        groupTypes={[
          {
            name: 'Priority tier',
            icon: 'star',
            color: '#059669',
            groups: ['vip'],
          },
          { name: 'Empty tier', icon: 'shield', color: '#2563eb', groups: [] },
        ]}
      />
    )
    await user.click(getTrigger())
    await user.type(screen.getByPlaceholderText('Search...'), 'not-found')
    expect(screen.getByText('No group found.')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Priority tier' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Empty tier' })).toBeVisible()
    await user.keyboard('{Escape}')
    expect(getTrigger()).toHaveFocus()
    expect(screen.getByTestId('selected-group')).toHaveTextContent('auto')
  })

  test('uses the compact table capsules in the selected group and dropdown options', () => {
    setReducedMotion(false)
    render(<Harness initialValue='auto' />)

    const trigger = getTrigger()
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(trigger).toHaveAttribute('data-auto-group-effect', 'trigger')
    expect(trigger).not.toHaveClass('bg-linear-to-r', 'overflow-hidden')
    expect(trigger).toHaveClass('overflow-visible')

    const triggerFlowBorder = trigger.querySelector<HTMLElement>(
      '[data-auto-group-flow-border]'
    )
    expect(triggerFlowBorder).toHaveAttribute('aria-hidden', 'true')
    expect(triggerFlowBorder).toHaveClass(
      'pointer-events-none',
      'auto-group-flow-border'
    )

    const triggerRatio = within(trigger)
      .getByText('Auto')
      .closest('[data-slot="badge"]')
    expect(triggerRatio).toHaveTextContent('Auto')
    expect(triggerRatio).not.toHaveTextContent('Ratio')
    expect(triggerRatio).not.toHaveTextContent('x')
    expect(trigger).not.toHaveTextContent('自动')
    expect(triggerRatio).toHaveClass(
      'relative',
      'overflow-visible',
      'rounded-md',
      'h-5',
      'min-w-12'
    )
    expect(
      triggerRatio?.querySelector('[data-auto-group-flow-border]')
    ).toHaveClass('auto-group-flow-border-subtle')

    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')

    const autoOption = getCommandItem('Global automatic routing')
    expect(autoOption).toHaveAttribute('data-auto-group-effect', 'option')
    expect(autoOption).toHaveAttribute('aria-selected', 'true')
    expect(autoOption).not.toHaveClass('bg-linear-to-r')
    expect(autoOption).toHaveClass('overflow-visible')
    expect(
      autoOption.querySelector('[data-auto-group-flow-border]')
    ).toBeInTheDocument()
    const optionRatio = within(autoOption)
      .getByText('Auto')
      .closest('[data-slot="badge"]')
    expect(optionRatio).toHaveTextContent('Auto')
    expect(optionRatio).not.toHaveTextContent('Ratio')
    expect(
      optionRatio?.querySelector('[data-auto-group-flow-border]')
    ).toHaveClass('auto-group-flow-border-subtle')

    const defaultOption = getCommandItem('User group')
    expect(defaultOption).not.toHaveAttribute('data-auto-group-effect')
    expect(defaultOption.querySelector('[data-auto-group-flow-border]')).toBe(
      null
    )
    const defaultRatio = within(defaultOption)
      .getByText('1x')
      .closest('[data-slot="badge"]')
    expect(defaultRatio).toHaveClass(
      'h-5',
      'min-w-12',
      'rounded-full',
      'tabular-nums',
      'border-muted-foreground/30'
    )
    expect(defaultRatio).not.toHaveTextContent('Ratio')
    expect(
      defaultOption.querySelector('[data-auto-group-effect="ratio"]')
    ).toBe(null)
  })

  test('keeps search and selection behavior while leaving normal groups unstyled', async () => {
    setReducedMotion(false)
    const { container } = render(<Harness initialValue='auto' />)

    const trigger = getTrigger()
    fireEvent.click(trigger)

    fireEvent.input(screen.getByPlaceholderText('Search...'), {
      target: { value: 'vip' },
    })

    const visibleOptions = [
      ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
    ]
    expect(
      visibleOptions.some((option) =>
        option.textContent?.includes('Global automatic routing')
      )
    ).toBe(false)
    const vipOption = getCommandItem('Priority group')
    fireEvent.click(vipOption)

    expect(within(container).getByTestId('selected-group')).toHaveTextContent(
      'vip'
    )
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(trigger).not.toHaveAttribute('data-auto-group-effect')
    expect(trigger.querySelector('[data-auto-group-flow-border]')).toBe(null)
  })

  test('renders group, top-up, and effective ratios as separate badges', () => {
    setReducedMotion(false)
    render(<Harness initialValue='vip' />)

    const trigger = getTrigger()
    expect(trigger).toHaveTextContent('3x Ratio')
    expect(trigger).toHaveTextContent('0.1x Top-up ratio')
    expect(trigger).toHaveTextContent('0.3x Effective ratio')
    expect(trigger.querySelectorAll('[data-slot="badge"]')).toHaveLength(3)
  })

  test('preserves the static Auto treatment but omits moving layers for reduced motion', async () => {
    setReducedMotion(true)
    render(<Harness initialValue='auto' />)

    const trigger = getTrigger()
    expect(trigger).toHaveAttribute('data-auto-group-effect', 'trigger')
    expect(trigger.querySelector('[data-auto-group-flow-border]')).toBe(null)
    expect(within(trigger).getByText('Auto')).toBeInTheDocument()

    fireEvent.click(trigger)
    const autoOption = getCommandItem('Global automatic routing')
    expect(autoOption).toHaveAttribute('data-auto-group-effect', 'option')
    expect(autoOption.querySelector('[data-auto-group-flow-border]')).toBe(null)
    expect(within(autoOption).getByText('Auto')).toBeInTheDocument()
    setReducedMotion(false)
  })
})
