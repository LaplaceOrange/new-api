import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it } from 'vitest'

import { formatUpstreamMoney } from '../../lib/format'
import type { Upstream } from '../../types'
import { UpstreamList } from '../upstream-list'
import { UpstreamStatus, UpstreamSummary } from '../upstream-summary'

function warningUpstream(): Upstream {
  return {
    id: 1,
    name: 'Low balance upstream',
    primary_url: 'https://upstream.example',
    addresses: ['https://upstream.example'],
    user_agent: '',
    auth_mode: 'jwt',
    auto_refresh_token: false,
    has_account_credentials: false,
    has_access_token: true,
    has_refresh_token: false,
    credential_blocked: false,
    balance: 4,
    balance_updated_at: 0,
    last_attempt_at: 0,
    last_error: '',
    created_at: 0,
    updated_at: 0,
    refreshing: false,
    channel_ids: [],
    snapshot: {
      days: 7,
      mode: 'rolling',
      start_at: 0,
      end_at: 0,
      timezone: 'Asia/Shanghai',
      consumption: 7,
      updated_at: 0,
      last_error: '',
      complete: true,
    },
    forecast: {
      status: 'warning',
      daily_consumption: 1,
      remaining_days: 4,
      suggested_topup: 3,
      stale: false,
    },
  }
}

afterEach(() => {
  cleanup()
  localStorage.clear()
})

it.each([true, false])(
  'shows exactly one suggestion next to the warning in a summary with compact=%s',
  (compact) => {
    render(<UpstreamSummary upstream={warningUpstream()} compact={compact} />)
    const suggestion = `Suggested top-up: ${formatUpstreamMoney(3)}`
    expect(screen.getAllByText(suggestion)).toHaveLength(1)
    expect(screen.getByText('Warning').parentElement).toContainElement(
      screen.getByText(suggestion)
    )
  }
)

it.each(['warning', 'critical', 'exhausted'] as const)(
  'shows the suggested amount for a %s balance',
  (status) => {
    const upstream = warningUpstream()
    upstream.forecast.status = status
    render(<UpstreamStatus upstream={upstream} />)
    expect(
      screen.getByText(`Suggested top-up: ${formatUpstreamMoney(3)}`)
    ).toBeVisible()
  }
)

it.each([
  'unknown',
  'healthy',
  'no_usage',
  'incomplete',
  'missing_balance',
  'zero',
  'invalid',
] as const)('does not show a suggestion when the forecast is %s', (state) => {
  const upstream = warningUpstream()
  if (state === 'unknown' || state === 'healthy' || state === 'no_usage') {
    upstream.forecast.status = state
  } else if (state === 'incomplete' && upstream.snapshot) {
    upstream.snapshot.complete = false
  } else if (state === 'missing_balance') {
    upstream.balance = null
  } else if (state === 'zero') {
    upstream.forecast.suggested_topup = 0
  } else {
    upstream.forecast.suggested_topup = Number.NaN
  }
  render(<UpstreamSummary upstream={upstream} compact />)
  expect(screen.queryByText(/Suggested top-up/)).not.toBeInTheDocument()
})

it('retains the last suggestion alongside the stale-data warning', () => {
  const upstream = warningUpstream()
  upstream.forecast.stale = true
  render(<UpstreamSummary upstream={upstream} compact />)
  expect(
    screen.getByText(`Suggested top-up: ${formatUpstreamMoney(3)}`)
  ).toBeVisible()
  expect(
    screen.getByText('Stale or incomplete data; estimates may be unavailable')
  ).toBeVisible()
})

it('keeps the suggestion visible after switching from card to table view', async () => {
  const upstream = warningUpstream()
  const router = createRouter({
    history: createMemoryHistory({ initialEntries: ['/'] }),
    routeTree: createRootRoute({
      component: () => (
        <UpstreamList
          items={[upstream]}
          loading={false}
          fetching={false}
          root={false}
          refreshPending={false}
          onRefresh={() => undefined}
          onEdit={() => undefined}
          onDelete={() => undefined}
        />
      ),
    }),
  })
  await router.load()
  render(<RouterProvider router={router} />)
  const suggestion = `Suggested top-up: ${formatUpstreamMoney(3)}`
  expect(await screen.findByText(suggestion)).toBeVisible()
  await userEvent
    .setup()
    .click(screen.getByRole('button', { name: 'Table view' }))
  expect(screen.getByRole('table')).toBeVisible()
  expect(screen.getByText(suggestion)).toBeVisible()
})
