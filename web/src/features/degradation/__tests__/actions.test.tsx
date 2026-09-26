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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { DegradationMonitor } from '../index'

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: React.ReactNode }) => (
    <div>{props.children}</div>
  ),
}))
vi.mock('../editor', () => ({
  DegradationEditor: () => null,
}))

const page = {
  success: true,
  data: {
    config: { enabled: true, groups: [] },
    groups: [
      {
        group: 'default',
        sort: 0,
        models: [
          {
            model: 'example-model',
            sort: 0,
            status: 'pending',
            detected_model: '',
            next_check_at: 0,
            timeline: [],
          },
        ],
      },
    ],
  },
}

function renderMonitor(role: number) {
  useAuthStore.getState().auth.setUser({ id: 1, username: 'viewer', role })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <DegradationMonitor />
    </QueryClientProvider>
  )
}

afterEach(() => {
  useAuthStore.getState().auth.setUser(null)
  vi.restoreAllMocks()
})

it('hides model actions from a non-admin viewer', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({ data: page })
  renderMonitor(1)

  expect(await screen.findByText('example-model')).toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: 'Open menu' })
  ).not.toBeInTheDocument()
})

it('opens admin actions and confirms deletion before clearing this model', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  vi.spyOn(api, 'get').mockResolvedValue({ data: page })
  const user = userEvent.setup()
  renderMonitor(10)

  await user.click(await screen.findByRole('button', { name: 'Open menu' }))
  expect(screen.getByRole('menuitem', { name: 'Test now' })).toBeInTheDocument()
  await user.click(screen.getByRole('menuitem', { name: 'Delete all records' }))
  expect(post).not.toHaveBeenCalled()
  await user.click(screen.getByRole('button', { name: 'Continue' }))
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/degradation/clear', {
      group: 'default',
      model: 'example-model',
    })
  )
})

it('starts a test for the selected model and refreshes after completion', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { task_id: 'systask-1' } },
  })
  const get = vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data:
      url === '/api/degradation/'
        ? page
        : {
            success: true,
            data: { status: 'succeeded', error: '' },
          },
  }))
  const user = userEvent.setup()
  renderMonitor(10)

  await user.click(await screen.findByRole('button', { name: 'Open menu' }))
  await user.click(screen.getByRole('menuitem', { name: 'Test now' }))
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/degradation/test', {
      group: 'default',
      model: 'example-model',
    })
  )
  await waitFor(() =>
    expect(get).toHaveBeenCalledWith('/api/degradation/test/systask-1')
  )
  await waitFor(() =>
    expect(
      get.mock.calls.filter(([url]) => url === '/api/degradation/').length
    ).toBeGreaterThan(1)
  )
})
