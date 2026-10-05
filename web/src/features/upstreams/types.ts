export type UpstreamPeriod = { days: 1 | 7 | 30; mode: 'rolling' | 'calendar' }

export type UpstreamConfig = {
  interval_minutes: number
  timezone: string
}

export type Upstream = {
  id: number
  name: string
  primary_url: string
  addresses: string[]
  user_agent: string
  auto_refresh_token: boolean
  credential_blocked: boolean
  balance: number | null
  balance_updated_at: number
  last_attempt_at: number
  last_error: string
  created_at: number
  updated_at: number
  has_access_token: boolean
  has_refresh_token: boolean
  refreshing: boolean
  channel_ids: number[]
  snapshot: {
    days: number
    mode: 'rolling' | 'calendar'
    start_at: number
    end_at: number
    timezone: string
    consumption: number | null
    updated_at: number
    last_error: string
    complete: boolean
  } | null
  forecast: {
    status:
      | 'unknown'
      | 'exhausted'
      | 'critical'
      | 'warning'
      | 'healthy'
      | 'no_usage'
    daily_consumption: number | null
    remaining_days: number | null
    suggested_topup: number | null
    stale: boolean
  }
}

export type UpstreamPayload = {
  name: string
  primary_url: string
  addresses: string[]
  user_agent: string
  auto_refresh_token: boolean
  access_token?: string
  refresh_token?: string
}

export type RankingOptions = UpstreamPeriod & {
  dimension: 'users' | 'models'
  metric: 'quota' | 'requests' | 'tokens'
  limit: 10 | 20 | 50
}

export type UpstreamRankings = {
  items: {
    user_id?: number
    name: string
    requests: number
    tokens: number
    quota: number
  }[]
  start_at: number
  end_at: number
  timezone: string
}
