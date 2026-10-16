import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import { UpstreamDetail } from '../detail'
import { Upstreams } from '../index'

afterEach(() => {
  cleanup()
  localStorage.clear()
})

it.each([
  { page: 'list', path: '/api/upstream/', component: Upstreams },
  {
    page: 'detail',
    path: '/api/upstream/1',
    component: () => <UpstreamDetail id={1} />,
  },
])(
  'defaults the $page to calendar and requests rolling data when selected',
  async ({ path, component }) => {
    const get = vi
      .spyOn(api, 'get')
      .mockImplementation(() => new Promise(() => undefined))
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    const router = createRouter({
      history: createMemoryHistory({ initialEntries: ['/'] }),
      routeTree: createRootRoute({ component }),
    })
    await router.load()
    render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    )
    expect(
      await screen.findByRole('tab', { name: 'Calendar' })
    ).toHaveAttribute('aria-selected', 'true')
    await waitFor(() =>
      expect(get).toHaveBeenCalledWith(
        path,
        expect.objectContaining({ params: { days: 7, mode: 'calendar' } })
      )
    )

    await userEvent.setup().click(screen.getByRole('tab', { name: 'Rolling' }))
    expect(screen.getByRole('tab', { name: 'Rolling' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
    await waitFor(() =>
      expect(get).toHaveBeenCalledWith(
        path,
        expect.objectContaining({ params: { days: 7, mode: 'rolling' } })
      )
    )
    queryClient.clear()
  }
)
