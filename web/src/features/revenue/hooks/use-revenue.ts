import { useQuery } from '@tanstack/react-query'

import { getRevenue } from '../api'
import type { RevenuePeriod } from '../types'

export function useRevenue(period: RevenuePeriod) {
  return useQuery({
    queryKey: ['revenue', period],
    queryFn: () => getRevenue(period),
    staleTime: 5 * 60 * 1000,
  })
}
