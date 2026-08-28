import { api } from '@/lib/api'

import type { RevenuePeriod, RevenueSnapshot } from './types'

type RevenueResponse = {
  success: boolean
  message?: string
  data: RevenueSnapshot
}

export async function getRevenue(period: RevenuePeriod): Promise<RevenueResponse> {
  const res = await api.get('/api/revenue', { params: { period } })
  return res.data
}
