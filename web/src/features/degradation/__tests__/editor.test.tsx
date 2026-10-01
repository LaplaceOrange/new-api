import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { DegradationEditor } from '../editor'
it('preserves a legacy expected name while adding and removing accepted versions', async () => {
  const user = userEvent.setup()
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: [] } })
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ data: { success: true } })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <DegradationEditor
        config={{
          enabled: true,
          interval_minutes: 60,
          retry_count: 2,
          retry_interval_minutes: 10,
          groups: [
            {
              group: 'default',
              sort: 0,
              models: [{ model: 'a', expected: 'model-v1', sort: 0 }],
            },
          ],
        }}
      />
    </QueryClientProvider>
  )
  await user.click(screen.getByRole('button', { name: 'Edit settings' }))
  const input = screen.getByRole('textbox', { name: 'Expected detected names' })
  await user.type(input, 'model-v2{Enter}')
  expect(screen.getByText('model-v1')).toBeInTheDocument()
  expect(screen.getByText('model-v2')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/degradation/',
      expect.objectContaining({
        groups: [
          {
            group: 'default',
            sort: 0,
            models: [{ model: 'a', expected: 'model-v1\nmodel-v2', sort: 0 }],
          },
        ],
      })
    )
  )
  await user.click(screen.getAllByRole('button', { name: 'Remove tag' })[0])
  expect(screen.queryByText('model-v1')).not.toBeInTheDocument()
})
