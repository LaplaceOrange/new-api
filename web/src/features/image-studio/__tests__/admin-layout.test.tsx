import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import { Main } from '@/components/layout/components/main'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ImageStudioAdmin } from '../admin'

vi.mock('../index', () => ({ GeneratedImage: () => null }))

afterEach(() => {
  cleanup()
  useAuthStore.getState().auth.setUser(null)
  vi.restoreAllMocks()
})

it('lets an admin scroll the settings page to the save action', async () => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: ROLE.ADMIN,
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      config: {
        operator_name: 'Operator',
        contact_email: 'contact@example.com',
        agreement: 'Agreement',
        privacy: 'Privacy',
        storage_mode: 'local',
        local_dir: '',
        s3_endpoint: '',
        s3_bucket: '',
        s3_region: '',
        retention_days: 30,
      },
      models: [],
      candidates: [],
    },
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })

  render(
    <QueryClientProvider client={client}>
      <Main className='p-0'>
        <ImageStudioAdmin />
      </Main>
    </QueryClientProvider>
  )

  expect(
    await screen.findByRole('button', { name: 'Save' })
  ).toBeInTheDocument()
  expect(
    screen.getByRole('region', { name: 'Image Studio Settings' })
  ).toHaveClass('overflow-y-auto')
})
