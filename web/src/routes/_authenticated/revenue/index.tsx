import { createFileRoute, redirect } from '@tanstack/react-router'
import z from 'zod'

import { Revenue } from '@/features/revenue'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

const revenueSearchSchema = z.object({
  period: z
    .enum(['today', 'week', 'month', 'year', 'all'])
    .optional()
    .catch(undefined),
})

export const Route = createFileRoute('/_authenticated/revenue/')({
  validateSearch: revenueSearchSchema,
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.ADMIN) {
      throw redirect({ to: '/403' })
    }
  },
  component: Revenue,
})
