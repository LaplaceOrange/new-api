import { api } from '@/lib/api'
import { authRequestOptions } from '@/lib/secure-verification'
import { requireServerSuccess } from '@/lib/server-error-message'

import type {
  RankingOptions,
  Upstream,
  UpstreamConfig,
  UpstreamPeriod,
  UpstreamRankings,
} from './types'

type Response<T> = { success: boolean; message?: string; data: T }

export async function getUpstreams(
  period: UpstreamPeriod,
  signal?: AbortSignal
) {
  const res = await api.get<
    Response<{ items: Upstream[]; config: UpstreamConfig }>
  >('/api/upstream/', { params: period, signal })
  return requireServerSuccess(res.data).data
}

export async function getUpstream(
  id: number,
  period: UpstreamPeriod,
  signal?: AbortSignal
) {
  const res = await api.get<Response<Upstream>>(`/api/upstream/${id}`, {
    params: period,
    signal,
  })
  return requireServerSuccess(res.data).data
}

export async function saveUpstream(
  id: number | null,
  serialized: string,
  proof: string,
  signal?: AbortSignal
) {
  const config = {
    ...authRequestOptions,
    signal,
    singleUseAuthorization: true,
    headers: { 'X-Security-Proof': proof, 'Content-Type': 'application/json' },
  }
  const res =
    id === null
      ? await api.post<Response<Upstream>>('/api/upstream/', serialized, config)
      : await api.put<Response<Upstream>>(
          `/api/upstream/${id}`,
          serialized,
          config
        )
  return requireServerSuccess(res.data).data
}

export async function deleteUpstream(id: number, proof: string) {
  const res = await api.delete<Response<unknown>>(`/api/upstream/${id}`, {
    ...authRequestOptions,
    singleUseAuthorization: true,
    headers: { 'X-Security-Proof': proof },
  })
  requireServerSuccess(res.data)
}

export async function refreshUpstreams(id?: number) {
  const res = await api.post<Response<unknown>>(
    id === undefined ? '/api/upstream/refresh' : `/api/upstream/${id}/refresh`
  )
  return requireServerSuccess(res.data).data
}

export async function getUpstreamConfig(signal?: AbortSignal) {
  const res = await api.get<Response<UpstreamConfig>>('/api/upstream/config', {
    signal,
  })
  return requireServerSuccess(res.data).data
}

export async function saveUpstreamConfig(
  payload: UpstreamConfig,
  proof: string,
  signal?: AbortSignal
) {
  const res = await api.put<Response<UpstreamConfig>>(
    '/api/upstream/config',
    payload,
    {
      ...authRequestOptions,
      signal,
      singleUseAuthorization: true,
      headers: { 'X-Security-Proof': proof },
    }
  )
  return requireServerSuccess(res.data).data
}

export async function getUpstreamRankings(
  id: number,
  options: RankingOptions,
  signal?: AbortSignal
) {
  const res = await api.get<Response<UpstreamRankings>>(
    `/api/upstream/${id}/rankings`,
    { params: options, signal }
  )
  return requireServerSuccess(res.data).data
}
