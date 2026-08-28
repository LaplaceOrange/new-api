export type RevenuePeriod = 'today' | 'week' | 'month' | 'year' | 'all'

export type RevenueSummary = {
  revenue: number
  orders: number
  customers: number
  average_order: number
  revenue_growth_pct: number
  order_growth_pct: number
}

export type RevenuePoint = {
  ts: string
  label: string
  revenue: number
  orders: number
  customers: number
}

export type RevenueProvider = {
  provider: string
  revenue: number
  orders: number
  customers: number
  share: number
}

export type RevenueCustomer = {
  rank: number
  user_id: number
  username: string
  display_name?: string
  revenue: number
  orders: number
}

export type RevenueSnapshot = {
  summary: RevenueSummary
  trend: RevenuePoint[]
  providers: RevenueProvider[]
  customers: RevenueCustomer[]
  customer_order_rankings: RevenueCustomer[]
}
