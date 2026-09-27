import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ImageStudioAdmin } from '../admin'
import { ImageStudio } from '../index'

vi.mock('@tanstack/react-router', () => ({
  Link: (props: { children: React.ReactNode }) => (
    <a href='#'>{props.children}</a>
  ),
}))

afterEach(() => {
  cleanup()
  useAuthStore.getState().auth.setUser(null)
  vi.restoreAllMocks()
})

it('shows per-request expression pricing in the admin candidates', async () => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'admin', role: ROLE.ADMIN })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      config: {
        operator_name: 'Operator',
        contact_email: 'contact@example.com',
        agreement: 'Agreement',
        privacy: 'Privacy',
        storage_mode: 'local',
        local_dir: '',
        retention_days: 30,
      },
      models: [],
      candidates: [
        {
          name: 'request-priced',
          price: 0.04,
          per_image: false,
          allow_edits: false,
          groups: { default: 1 },
        },
      ],
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  render(
    <QueryClientProvider client={client}>
      <ImageStudioAdmin />
    </QueryClientProvider>
  )

  expect(await screen.findByText('$0.04 / request')).toBeInTheDocument()
  expect(
    screen.getByRole('checkbox', { name: 'request-priced' })
  ).toBeInTheDocument()
})

it('keeps a per-request estimate unchanged when requesting multiple images', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (path) => ({
    data:
      path === '/api/image-studio/options'
        ? {
            ready: true,
            models: [
              {
                name: 'request-priced',
                price: 0.04,
                per_image: false,
                allow_edits: false,
                groups: { default: 1 },
              },
            ],
            agreement_revision: 'agreement',
            privacy_revision: 'privacy',
          }
        : { data: [], total: 0 },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <ImageStudio />
    </QueryClientProvider>
  )
  const estimate = screen.getByText('Estimated cost').parentElement
  expect(estimate).not.toBeNull()
  if (!estimate) return
  expect(await within(estimate).findByText('$0.0400')).toBeInTheDocument()

  await userEvent.type(
    screen.getByRole('spinbutton', { name: 'Number of images' }),
    '3'
  )

  expect(within(estimate).getByText('$0.0400')).toBeInTheDocument()
})

it('keeps per-image estimates for models without the new pricing unit field', async () => {
  vi.spyOn(api, 'get').mockImplementation(async (path) => ({
    data:
      path === '/api/image-studio/options'
        ? {
            ready: true,
            models: [
              {
                name: 'legacy-image',
                price: 0.02,
                allow_edits: false,
                groups: { default: 1 },
              },
            ],
            agreement_revision: 'agreement',
            privacy_revision: 'privacy',
          }
        : { data: [], total: 0 },
  }))
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <ImageStudio />
    </QueryClientProvider>
  )
  const estimate = screen.getByText('Estimated cost').parentElement
  expect(estimate).not.toBeNull()
  if (!estimate) return
  expect(await within(estimate).findByText('$0.0200')).toBeInTheDocument()

  await userEvent.type(
    screen.getByRole('spinbutton', { name: 'Number of images' }),
    '3'
  )

  expect(within(estimate).getByText('$0.0600')).toBeInTheDocument()
})
