import { createFileRoute, redirect } from '@tanstack/react-router'

import { UpstreamDetail } from '@/features/upstreams/detail'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/upstreams/$id')({
  beforeLoad: ({ params }) => {
    if ((useAuthStore.getState().auth.user?.role ?? 0) < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
    if (
      !/^[1-9]\d*$/.test(params.id) ||
      !Number.isSafeInteger(Number(params.id))
    ) {
      throw redirect({ to: '/404' })
    }
  },
  component: UpstreamDetailRoute,
})

function UpstreamDetailRoute() {
  const { id } = Route.useParams()
  return <UpstreamDetail id={Number(id)} />
}
