import { createFileRoute, redirect } from '@tanstack/react-router'

import { Upstreams } from '@/features/upstreams'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/upstreams/')({
  beforeLoad: () => {
    if ((useAuthStore.getState().auth.user?.role ?? 0) < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  component: Upstreams,
})
