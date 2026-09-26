import { createFileRoute, redirect } from '@tanstack/react-router'

import { Main } from '@/components/layout'
import { ImageStudioAdmin } from '@/features/image-studio/admin'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/image-studio-admin/')({
  beforeLoad: () => {
    if ((useAuthStore.getState().auth.user?.role ?? 0) < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  component: () => <Main className='p-0'><ImageStudioAdmin /></Main>,
})
