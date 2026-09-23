/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { getAllUserCleanupCandidates, getAllUserCleanupHistory } from '../api'
import { UserCleanupPage } from '../index'
import { formatCleanupCsv } from '../lib'

vi.mock('@/features/users/components/users-columns', () => ({
  useUsersColumns: () => [
    {
      id: 'select',
      header: 'Select',
      cell: ({
        row,
      }: {
        row: { getIsSelected(): boolean; toggleSelected(value: boolean): void }
      }) => (
        <input
          aria-label='Select row'
          type='checkbox'
          checked={row.getIsSelected()}
          onChange={(event) => row.toggleSelected(event.target.checked)}
        />
      ),
    },
    {
      accessorKey: 'username',
      header: 'Username',
      cell: ({ row }: { row: { original: { username: string } } }) =>
        row.original.username,
    },
  ],
}))

vi.mock('@/components/layout', () => {
  const Slot = (props: { children: React.ReactNode }) => (
    <div>{props.children}</div>
  )
  const Layout = (props: { children: React.ReactNode }) => (
    <div>{props.children}</div>
  )
  Object.assign(Layout, { Title: Slot, Content: Slot })
  return { SectionPageLayout: Layout }
})

vi.mock('@/components/group-badge', () => ({
  GroupBadge: (props: { group: string }) => <span>{props.group}</span>,
}))

vi.mock('@/components/status-badge', () => ({
  StatusBadge: (props: { label: string }) => <span>{props.label}</span>,
}))

vi.mock('@/components/table-id', () => ({
  TableId: (props: { value: number }) => <span>{props.value}</span>,
}))

function pageResponse<T>(
  items: T[],
  total: number,
  page: number,
  pageSize = 100
) {
  return {
    success: true,
    data: { items, total, page, page_size: pageSize },
  }
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

test('loads every candidate and history page in bounded batches', async () => {
  const users = Array.from({ length: 205 }, (_, index) => ({
    id: index + 1,
    username: `user-${index + 1}`,
  }))
  const records = Array.from({ length: 205 }, (_, index) => ({
    id: index + 1,
    user_id: index + 1,
    username: `history-${index + 1}`,
    display_name: '',
    status: 2,
    group: 'default',
    quota: 0,
    cleaned_at: 1_700_000_000,
  }))
  const get = vi.spyOn(api, 'get').mockImplementation(async (url, config) => {
    const page = Number(
      (config as { params?: { p?: number } } | undefined)?.params?.p ?? 1
    )
    if (url === '/api/user/cleanup/candidates') {
      return {
        data: pageResponse(
          users.slice((page - 1) * 100, page * 100),
          205,
          page
        ),
      }
    }
    if (url === '/api/user/cleanup/history') {
      return {
        data: pageResponse(
          records.slice((page - 1) * 100, page * 100),
          205,
          page
        ),
      }
    }
    throw new Error(`Unexpected GET ${url}`)
  })

  const [candidateResult, historyResult] = await Promise.all([
    getAllUserCleanupCandidates(),
    getAllUserCleanupHistory(),
  ])

  expect(candidateResult.data?.items).toHaveLength(205)
  expect(candidateResult.data?.items.at(-1)?.username).toBe('user-205')
  expect(historyResult.data?.items).toHaveLength(205)
  expect(historyResult.data?.items.at(-1)?.username).toBe('history-205')
  expect(get).toHaveBeenCalledTimes(6)
  for (const call of get.mock.calls) {
    expect(call[1]?.params?.page_size).toBe(100)
  }
})

test('quotes CSV values and protects formula-like strings', () => {
  expect(
    formatCleanupCsv(
      ['name', 'value'],
      [
        ['A "quoted", name', '=1+1'],
        ['line\nbreak', -12],
      ]
    )
  ).toBe(
    '"name","value"\r\n"A ""quoted"", name","\'=1+1"\r\n"line\nbreak","-12"\r\n'
  )
})

test('shows all loaded rows, removes history pagination, and exports selected users and history', async () => {
  const downloads: { filename: string; blob: Blob }[] = []
  let downloadedBlob: Blob
  vi.stubGlobal(
    'URL',
    Object.assign(class extends URL {}, {
      createObjectURL: vi.fn((blob: Blob) => {
        downloadedBlob = blob
        return 'blob:user-cleanup-export'
      }),
      revokeObjectURL: vi.fn(),
    })
  )
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(
    function (this: HTMLAnchorElement) {
      downloads.push({ filename: this.download, blob: downloadedBlob })
    }
  )

  const candidates = Array.from({ length: 21 }, (_, index) => ({
    id: index + 1,
    username: `candidate-${index + 1}`,
    display_name: `Display ${index + 1}`,
    email: `candidate-${index + 1}@example.com`,
    status: 1,
    role: 1,
    group: 'default',
    quota: 500,
    used_quota: 250,
    request_count: 3,
    created_at: 1_700_000_000,
    last_login_at: 1_700_000_100,
    remark: '',
  }))
  const records = Array.from({ length: 21 }, (_, index) => ({
    id: index + 1,
    user_id: index + 1,
    username: `cleaned-${index + 1}`,
    display_name: '',
    status: 2,
    group: 'default',
    quota: 0,
    cleaned_at: 1_700_000_000,
  }))
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/user/cleanup/rules') {
      return {
        data: {
          success: true,
          data: { rules: [], unmatched_action: 'ignore' },
        },
      }
    }
    if (url === '/api/user/cleanup/scan') {
      return { data: { success: true, data: null } }
    }
    if (url === '/api/group/') {
      return { data: { success: true, data: ['default'] } }
    }
    if (url === '/api/user/cleanup/candidates') {
      return { data: pageResponse(candidates, 21, 1) }
    }
    if (url === '/api/user/cleanup/history') {
      return { data: pageResponse(records, 21, 1) }
    }
    throw new Error(`Unexpected GET ${url}`)
  })

  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const actor = userEvent.setup()
  render(
    <QueryClientProvider client={client}>
      <UserCleanupPage />
    </QueryClientProvider>
  )

  expect(await screen.findByText('candidate-21')).toBeInTheDocument()
  expect(screen.getByText('cleaned-21')).toBeInTheDocument()
  expect(
    screen.queryByRole('navigation', { name: 'Page' })
  ).not.toBeInTheDocument()
  await actor.click(screen.getAllByRole('checkbox', { name: 'Select row' })[0])
  await actor.click(screen.getByRole('button', { name: 'Export selected CSV' }))
  await actor.click(
    screen.getByRole('button', { name: 'Export cleanup history CSV' })
  )

  expect(downloads).toHaveLength(2)
  expect(downloads[0].filename).toMatch(/^user-cleanup-selected-\d+\.csv$/)
  expect(downloads[1].filename).toMatch(/^user-cleanup-history-\d+\.csv$/)
  expect(await downloads[0].blob.text()).toContain('candidate-1')
  expect(await downloads[1].blob.text()).toContain('cleaned-21')
  client.clear()
})
