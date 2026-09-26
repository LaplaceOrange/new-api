import { api } from '@/lib/api'

export type StudioModel = {
  name: string
  allow_edits: boolean
  price: number
  groups: Record<string, number>
}

export type StudioOptions = {
  ready: boolean
  models: StudioModel[]
  operator_name: string
  contact_email: string
  agreement_revision: string
  privacy_revision: string
  retention_days: number
}

export type StudioAsset = {
  id: number
  kind: 'reference' | 'result'
  mime_type: string
}

export type StudioRecord = {
  id: number
  user_id: number
  model: string
  group: string
  prompt: string
  size: string
  count: number
  estimated_price: number
  actual_price?: number | null
  status: 'processing' | 'completed' | 'failed' | 'expired'
  error?: string
  created_at: string
  expires_at: string
  assets: StudioAsset[]
}

export type StudioLegal = {
  content: string
  revision: string
  operator_name: string
  contact_email: string
}

export type StudioAdminConfig = {
  operator_name: string
  contact_email: string
  agreement: string
  privacy: string
  storage_mode: 'local' | 's3'
  local_dir: string
  s3_endpoint: string
  s3_bucket: string
  s3_region: string
  s3_access_key?: string
  s3_secret_key?: string
  retention_days: number
}

export async function studioAsset(recordId: number, assetId: number, admin = false): Promise<string> {
  const base = admin ? '/api/image-studio/admin' : '/api/image-studio'
  const response = await api.get(`${base}/records/${recordId}/assets/${assetId}`, { responseType: 'blob' })
  return URL.createObjectURL(response.data as Blob)
}

export async function studioDownload(recordId: number, assetId: number, admin = false): Promise<void> {
  const url = await studioAsset(recordId, assetId, admin)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = `image-${recordId}-${assetId}`
  anchor.click()
  window.setTimeout(() => URL.revokeObjectURL(url), 60000)
}
