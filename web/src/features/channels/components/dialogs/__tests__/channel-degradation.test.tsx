import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { channelSchema } from '../../../types'
import { ChannelDegradationDialog } from '../channel-degradation-dialog'
import { ChannelTestDialogContent } from '../channel-test-dialog'

const channel = channelSchema.parse({
  id: 42,
  type: 1,
  key: '',
  status: 1,
  name: 'Channel A',
  models: 'model-a,model-b',
  group: 'default',
  created_time: 0,
  test_time: 0,
  response_time: 0,
  balance_updated_time: 0,
})
const originalAuth = useAuthStore.getState().auth
beforeEach(() => {
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'admin',
    role: 100,
    status: 1,
    group: 'default',
  })
})
afterEach(() => {
  useAuthStore.setState({ auth: originalAuth })
})
function renderDialog(models = channel.models) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ChannelDegradationDialog
        channel={{ ...channel, models }}
        open
        onOpenChange={() => {}}
      />
    </QueryClientProvider>
  )
}
it('lists optional expected names after model names and tests all models independently', async () => {
  const user = userEvent.setup()
  const post = vi
    .spyOn(api, 'post')
    .mockImplementation(async (_url, input) => ({
      data: {
        success: true,
        data: { task_id: (input as { model: string }).model },
      },
    }))
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data: {
        status: 'succeeded',
        error: '',
        result: {
          status: url.endsWith('model-a') ? 'passed' : 'suspected',
          detected_model: url.endsWith('model-a') ? 'Model A' : 'Model C',
          model: url.split('/').at(-1),
          channel_id: 42,
        },
      },
    },
  }))
  renderDialog()
  const headers = screen
    .getAllByRole('columnheader')
    .map((item) => item.textContent)
  expect(headers.indexOf('Expected detected name')).toBe(
    headers.indexOf('Model') + 1
  )
  await user.type(
    screen.getByRole('textbox', { name: 'Expected detected name · model-b' }),
    'Model B1,Model B2'
  )
  await user.click(screen.getByRole('button', { name: 'Test all 2 models' }))
  await waitFor(() => expect(post).toHaveBeenCalledTimes(2))
  expect(post).toHaveBeenCalledWith('/api/channel/42/degradation', {
    group: 'default',
    model: 'model-a',
  })
  expect(post).toHaveBeenCalledWith('/api/channel/42/degradation', {
    group: 'default',
    model: 'model-b',
    expected: 'Model B1\nModel B2',
  })
  expect(await screen.findByText('Detected model: Model A')).toBeInTheDocument()
  expect(
    await screen.findByText('Suspected degradation: Model C')
  ).toBeInTheDocument()
  expect(
    screen.getByRole('button', { name: 'Test all 2 models' })
  ).toBeEnabled()
})
it('runs only the requested row when its test button is clicked', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true, data: { task_id: 'single' } } })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        status: 'succeeded',
        result: {
          status: 'passed',
          detected_model: 'B',
          model: 'model-b',
          channel_id: 42,
        },
      },
    },
  })
  renderDialog()
  await userEvent
    .setup()
    .click(screen.getByRole('button', { name: 'Test degradation · model-b' }))
  expect(await screen.findByText('Detected model: B')).toBeInTheDocument()
  expect(post).toHaveBeenCalledTimes(1)
  expect(post).toHaveBeenCalledWith('/api/channel/42/degradation', {
    group: 'default',
    model: 'model-b',
  })
})
it('shows a failed row without blocking another model and permits a new attempt after a missing task', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { task_id: 'missing-task' } },
  })
  const get = vi
    .spyOn(api, 'get')
    .mockResolvedValue({ data: { success: false, message: 'test not found' } })
  renderDialog()
  const user = userEvent.setup()
  await user.click(
    screen.getByRole('button', { name: 'Test degradation · model-a' })
  )
  expect(await screen.findByText('test not found')).toBeInTheDocument()
  expect(
    screen.getByRole('button', { name: 'Test degradation · model-b' })
  ).toBeEnabled()
  get.mockResolvedValue({
    data: {
      success: true,
      data: {
        status: 'succeeded',
        result: {
          status: 'passed',
          detected_model: 'A',
          model: 'model-a',
          channel_id: 42,
        },
      },
    },
  })
  await user.click(
    screen.getByRole('button', { name: 'Test degradation · model-a' })
  )
  expect(await screen.findByText('Detected model: A')).toBeInTheDocument()
  expect(post).toHaveBeenCalledTimes(2)
})
it('disables testing when the channel has no models', () => {
  renderDialog('')
  expect(
    screen.getByText('This channel has no configured models.')
  ).toBeInTheDocument()
  expect(
    screen.getByRole('button', { name: 'Test all 0 models' })
  ).toBeDisabled()
})
it('preserves the connection test layout without expected-name fields', () => {
  const client = new QueryClient()
  render(
    <QueryClientProvider client={client}>
      <ChannelTestDialogContent
        currentRow={channel}
        open
        onOpenChange={() => {}}
      />
    </QueryClientProvider>
  )
  expect(
    screen.queryByRole('columnheader', { name: 'Expected detected name' })
  ).not.toBeInTheDocument()
  expect(screen.getByText('Endpoint Type')).toBeInTheDocument()
  expect(screen.getByText('Stream Mode')).toBeInTheDocument()
  expect(
    within(screen.getByRole('region', { name: 'Channel models' })).getAllByRole(
      'button',
      { name: 'Test Connection' }
    )
  ).toHaveLength(2)
})
it('locks submitted expectations while resuming a task after a network error', async () => {
  const post = vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { task_id: 'resume-task' } },
  })
  const get = vi
    .spyOn(api, 'get')
    .mockRejectedValueOnce(new Error('Network unavailable'))
  renderDialog()
  const user = userEvent.setup()
  const input = screen.getByRole('textbox', {
    name: 'Expected detected name · model-a',
  })
  await user.type(input, 'Model A')
  await user.click(
    screen.getByRole('button', { name: 'Test degradation · model-a' })
  )
  expect(await screen.findByText('Network unavailable')).toBeInTheDocument()
  expect(
    screen.getByRole('textbox', { name: 'Expected detected name · model-a' })
  ).toBeDisabled()
  expect(
    screen.getByRole('textbox', { name: 'Expected detected name · model-a' })
  ).toHaveValue('Model A')
  get.mockResolvedValue({
    data: {
      success: true,
      data: {
        status: 'succeeded',
        result: {
          status: 'passed',
          detected_model: 'Model A',
          model: 'model-a',
          channel_id: 42,
        },
      },
    },
  })
  await user.click(
    screen.getByRole('button', { name: 'Test degradation · model-a' })
  )
  expect(await screen.findByText('Detected model: Model A')).toBeInTheDocument()
  expect(post).toHaveBeenCalledTimes(1)
  expect(
    screen.getByRole('textbox', { name: 'Expected detected name · model-a' })
  ).toBeEnabled()
})

it.each([false, true])(
  'lists disabled models and bulk updates only completed results in degradation=%s',
  async (degradation) => {
    const user = userEvent.setup()
    const post = vi
      .spyOn(api, 'post')
      .mockImplementation(async (url, input) => {
        if (url.endsWith('/models/status')) {
          const payload = input as { models: string[]; enabled: boolean }
          return {
            data: {
              success: true,
              data: {
                disabled_models: payload.enabled ? {} : { 'model-b': true },
              },
            },
          }
        }
        return {
          data: {
            success: true,
            data: { task_id: (input as { model: string }).model },
          },
        }
      })
    vi.spyOn(api, 'get').mockImplementation(async (url, config) => {
      if (url.startsWith('/api/channel/test/')) {
        const model = config?.params.model
        return {
          data: {
            success: model === 'model-a',
            message: 'upstream failed',
            time: 0.1,
          },
        }
      }
      return {
        data: {
          success: true,
          data: {
            status: 'succeeded',
            result: {
              status: url.endsWith('model-a') ? 'passed' : 'suspected',
              detected_model: 'detected',
              model: url.split('/').at(-1),
              channel_id: 42,
            },
          },
        },
      }
    })
    render(
      <QueryClientProvider client={new QueryClient()}>
        <ChannelTestDialogContent
          currentRow={{
            ...channel,
            models: 'model-a,model-b,untested',
            channel_info: {
              ...channel.channel_info,
              disabled_models: { 'model-a': true },
            },
          }}
          open
          onOpenChange={() => {}}
          degradation={degradation}
        />
      </QueryClientProvider>
    )
    expect(
      screen.getByRole('switch', { name: 'Enable routing for model-a' })
    ).not.toBeChecked()
    const region = within(
      screen.getByRole('region', { name: 'Channel models' })
    )
    const testButtons = region.getAllByRole('button', {
      name: degradation ? /^Test degradation/ : 'Test Connection',
    })
    await user.click(testButtons[0])
    await screen.findByRole('button', { name: 'Enable successful models (1)' })
    await user.click(
      region.getAllByRole('button', {
        name: degradation ? /^Test degradation/ : 'Test Connection',
      })[1]
    )
    await screen.findByRole('button', { name: 'Disable failed models (1)' })
    await user.click(
      screen.getByRole('button', { name: 'Enable successful models (1)' })
    )
    await waitFor(() =>
      expect(
        screen.getByRole('switch', { name: 'Enable routing for model-a' })
      ).toBeChecked()
    )
    await user.click(
      screen.getByRole('button', { name: 'Disable failed models (1)' })
    )
    await waitFor(() =>
      expect(
        screen.getByRole('switch', { name: 'Enable routing for model-b' })
      ).not.toBeChecked()
    )
    expect(post).toHaveBeenCalledWith(
      '/api/channel/42/models/status',
      { models: ['model-a'], enabled: true },
      expect.anything()
    )
    expect(post).toHaveBeenCalledWith(
      '/api/channel/42/models/status',
      { models: ['model-b'], enabled: false },
      expect.anything()
    )
    expect(
      screen.getByRole('switch', { name: 'Enable routing for untested' })
    ).toBeChecked()
    expect(region.getByText('model-b')).toBeInTheDocument()
  }
)

it('preserves routing state after a rejected toggle and allows retry', async () => {
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValueOnce({
      data: { success: false, message: 'cannot save' },
    })
    .mockResolvedValueOnce({
      data: { success: true, data: { disabled_models: { 'model-a': true } } },
    })
  renderDialog()
  const toggle = screen.getByRole('switch', {
    name: 'Enable routing for model-a',
  })
  const user = userEvent.setup()
  await user.click(toggle)
  await waitFor(() =>
    expect(
      screen.getByRole('switch', { name: 'Enable routing for model-a' })
    ).not.toHaveAttribute('aria-disabled', 'true')
  )
  expect(
    screen.getByRole('switch', { name: 'Enable routing for model-a' })
  ).toBeChecked()
  await user.click(
    screen.getByRole('switch', { name: 'Enable routing for model-a' })
  )
  await waitFor(() =>
    expect(
      screen.getByRole('switch', { name: 'Enable routing for model-a' })
    ).not.toBeChecked()
  )
  expect(post).toHaveBeenCalledTimes(2)
})

it('disables all routing switches while a save is pending', async () => {
  let finish!: (value: unknown) => void
  vi.spyOn(api, 'post').mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve
      })
  )
  renderDialog()
  await userEvent
    .setup()
    .click(screen.getByRole('switch', { name: 'Enable routing for model-a' }))
  expect(
    screen.getByRole('switch', { name: 'Enable routing for model-a' })
  ).toHaveAttribute('aria-disabled', 'true')
  expect(
    screen.getByRole('switch', { name: 'Enable routing for model-b' })
  ).toHaveAttribute('aria-disabled', 'true')
  finish({
    data: { success: true, data: { disabled_models: { 'model-a': true } } },
  })
  await waitFor(() =>
    expect(
      screen.getByRole('switch', { name: 'Enable routing for model-b' })
    ).not.toHaveAttribute('aria-disabled', 'true')
  )
})

it('prevents routing changes without channel operation permission', () => {
  useAuthStore.getState().auth.setUser(null)
  renderDialog()
  expect(
    screen.getByRole('switch', { name: 'Enable routing for model-a' })
  ).toHaveAttribute('aria-disabled', 'true')
})
