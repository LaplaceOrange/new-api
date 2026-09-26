import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, Eye, Save, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { studioDownload, type StudioAdminConfig, type StudioModel, type StudioRecord } from './api'
import { GeneratedImage } from './index'

type AdminResponse = {
  config: StudioAdminConfig
  models: Pick<StudioModel, 'name' | 'allow_edits'>[]
  candidates: StudioModel[]
}

export function ImageStudioAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isRoot = (useAuthStore((state) => state.auth.user?.role) ?? 0) >= ROLE.SUPER_ADMIN
  const [view, setView] = useState<'settings' | 'records'>('settings')
  const [config, setConfig] = useState<StudioAdminConfig | null>(null)
  const [models, setModels] = useState<Pick<StudioModel, 'name' | 'allow_edits'>[]>([])
  const [preview, setPreview] = useState<'agreement' | 'privacy' | null>(null)
  const [saving, setSaving] = useState(false)
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')
  const [userId, setUserId] = useState('')
  const [selected, setSelected] = useState<number[]>([])
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [previewAsset, setPreviewAsset] = useState<{ recordId: number; assetId: number } | null>(null)
  const settings = useQuery({
    queryKey: ['image-studio', 'admin', 'config'],
    queryFn: async () => (await api.get<AdminResponse>('/api/image-studio/admin/config')).data,
  })
  const records = useQuery({
    queryKey: ['image-studio', 'admin', 'records', page, status, userId],
    queryFn: async () => (await api.get<{ data: StudioRecord[]; total: number }>('/api/image-studio/admin/records', {
      params: { page, status, user_id: userId },
    })).data,
    enabled: view === 'records',
  })
  useEffect(() => {
    if (!settings.data || config) return
    setConfig(settings.data.config)
    setModels(settings.data.models)
  }, [settings.data, config])

  function update<K extends keyof StudioAdminConfig>(key: K, value: StudioAdminConfig[K]) {
    setConfig((current) => current ? { ...current, [key]: value } : current)
  }
  function toggleModel(name: string) {
    setModels((current) => current.some((item) => item.name === name)
      ? current.filter((item) => item.name !== name)
      : [...current, { name, allow_edits: false }])
  }
  async function save() {
    if (!config) return
    setSaving(true)
    try {
      await api.put('/api/image-studio/admin/config', { ...config, models })
      toast.success(t('Settings saved'))
      setConfig((current) => current ? { ...current, s3_access_key: '', s3_secret_key: '' } : current)
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'admin', 'config'] })
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'options'] })
    } catch { toast.error(t('Failed to save settings')) }
    finally { setSaving(false) }
  }
  async function batch(action: 'delete' | 'zip', ids = selected) {
    if (!ids.length) return
    try {
      const response = await api.post(`/api/image-studio/admin/records/${action}`, { ids }, action === 'zip' ? { responseType: 'blob' } : undefined)
      if (action === 'zip') {
        const url = URL.createObjectURL(response.data as Blob)
        const anchor = document.createElement('a')
        anchor.href = url
        anchor.download = 'image-studio.zip'
        anchor.click()
        window.setTimeout(() => URL.revokeObjectURL(url), 60000)
      } else {
        setSelected([])
        setConfirmDelete(false)
        await queryClient.invalidateQueries({ queryKey: ['image-studio', 'admin', 'records'] })
      }
    } catch { toast.error(t('Operation failed')) }
  }

  return (
    <div className='mx-auto max-w-7xl space-y-5 px-5 py-6'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <h1 className='text-xl font-semibold'>{t('Image Studio Settings')}</h1>
        <div className='flex gap-2'>
          <Button variant={view === 'settings' ? 'secondary' : 'ghost'} onClick={() => setView('settings')}>{t('Settings')}</Button>
          <Button variant={view === 'records' ? 'secondary' : 'ghost'} onClick={() => setView('records')}>{t('All generations')}</Button>
        </div>
      </div>
      {view === 'settings' && config && <>
        <section className='space-y-4 border-t pt-5'>
          <h2 className='text-sm font-semibold'>{t('Operator details')}</h2>
          <div className='grid gap-3 sm:grid-cols-2'>
            <label className='text-sm'>{t('Operator name')}<Input className='mt-2' value={config.operator_name} onChange={(event) => update('operator_name', event.target.value)} /></label>
            <label className='text-sm'>{t('Contact email')}<Input className='mt-2' type='email' value={config.contact_email} onChange={(event) => update('contact_email', event.target.value)} /></label>
          </div>
          <p className='text-muted-foreground text-xs'>{t('Generation requires an operator and contact email.')}</p>
        </section>
        <section className='space-y-3 border-t pt-5'>
          <h2 className='text-sm font-semibold'>{t('Available image models')}</h2>
          <div className='grid gap-2 md:grid-cols-2 xl:grid-cols-3'>
            {settings.data?.candidates.map((item) => {
              const enabled = models.some((entry) => entry.name === item.name)
              return <div key={item.name} className='flex min-w-0 items-center gap-3 border p-3'>
                <Checkbox checked={enabled} onCheckedChange={() => toggleModel(item.name)} aria-label={item.name} />
                <div className='min-w-0 flex-1'>
                  <span className='block truncate text-sm font-medium'>{item.name}</span>
                  <span className='text-muted-foreground text-xs'>${item.price} / {t('image')}</span>
                </div>
                {enabled && <label className='flex items-center gap-2 text-xs'>
                  <Checkbox checked={models.find((entry) => entry.name === item.name)?.allow_edits ?? false} onCheckedChange={(value) => setModels((current) => current.map((entry) => entry.name === item.name ? { ...entry, allow_edits: value === true } : entry))} />
                  {t('Reference images')}
                </label>}
              </div>
            })}
          </div>
          {!settings.data?.candidates.length && <p className='text-muted-foreground text-sm'>{t('No fixed-price image models available')}</p>}
        </section>
        <section className='grid gap-5 border-t pt-5 xl:grid-cols-2'>
          {(['agreement', 'privacy'] as const).map((kind) => <div key={kind} className='space-y-2'>
            <div className='flex items-center justify-between'>
              <h2 className='text-sm font-semibold'>{t(kind === 'agreement' ? 'Image Studio User Agreement' : 'Image Studio Privacy Policy')}</h2>
              <Button variant='ghost' size='sm' onClick={() => setPreview(preview === kind ? null : kind)}>{t('Preview')}</Button>
            </div>
            <Textarea className='min-h-64 font-mono text-xs' value={config[kind]} onChange={(event) => update(kind, event.target.value)} />
            {preview === kind && <div className='max-h-80 overflow-y-auto border p-4 text-sm'>
              <p className='text-muted-foreground mb-3 text-xs'>{config.operator_name} · {config.contact_email}</p>
              <RichContent mode='markdown' content={config[kind]} />
            </div>}
          </div>)}
        </section>
        {isRoot && <section className='space-y-3 border-t pt-5'>
          <h2 className='text-sm font-semibold'>{t('Private image storage')}</h2>
          <div className='grid gap-3 sm:grid-cols-2'>
            <label className='text-sm'>{t('Storage mode')}
              <NativeSelect className='mt-2 w-full' value={config.storage_mode} onChange={(event) => update('storage_mode', event.target.value as 'local' | 's3')}>
                <NativeSelectOption value='local'>{t('Local storage')}</NativeSelectOption>
                <NativeSelectOption value='s3'>{t('S3-compatible storage')}</NativeSelectOption>
              </NativeSelect>
            </label>
            <label className='text-sm'>{t('Retention (days)')}<Input className='mt-2' type='number' min='1' max='180' value={config.retention_days} onChange={(event) => update('retention_days', Number(event.target.value))} /></label>
            {config.storage_mode === 'local' && <label className='text-sm'>{t('Storage directory')}<Input className='mt-2' value={config.local_dir} onChange={(event) => update('local_dir', event.target.value)} /></label>}
            {config.storage_mode === 's3' && <>
              {(['s3_endpoint', 's3_bucket', 's3_region', 's3_access_key', 's3_secret_key'] as const).map((field) => <label key={field} className='text-sm'>{t(field)}
                <Input className='mt-2' type={field === 's3_secret_key' ? 'password' : 'text'} value={config[field] ?? ''} onChange={(event) => update(field, event.target.value)} autoComplete='off' />
              </label>)}
            </>}
          </div>
        </section>}
        <div className='flex justify-end border-t pt-4'><Button disabled={saving} onClick={() => void save()}><Save />{t('Save')}</Button></div>
      </>}
      {view === 'records' && <>
        <div className='flex flex-wrap items-center gap-2 border-t pt-5'>
          <Input className='w-32' type='number' placeholder={t('User ID')} value={userId} onChange={(event) => { setUserId(event.target.value); setPage(1) }} />
          <NativeSelect value={status} onChange={(event) => { setStatus(event.target.value); setPage(1) }}>
            <NativeSelectOption value=''>{t('All statuses')}</NativeSelectOption>
            {['completed', 'processing', 'failed', 'expired'].map((value) => <NativeSelectOption key={value} value={value}>{t(value)}</NativeSelectOption>)}
          </NativeSelect>
          <span className='text-muted-foreground ml-auto text-xs'>{selected.length} {t('selected')}</span>
          <Button variant='outline' size='sm' disabled={!selected.length} onClick={() => void batch('zip')}><Download />{t('Download ZIP')}</Button>
          <Button variant='destructive' size='sm' disabled={!selected.length} onClick={() => setConfirmDelete(true)}><Trash2 />{t('Delete')}</Button>
        </div>
        <div className='divide-y border-y'>
          {(records.data?.data ?? []).map((record) => <div key={record.id} className='flex min-w-0 items-start gap-3 py-4'>
            <Checkbox checked={selected.includes(record.id)} onCheckedChange={(value) => setSelected((current) => value === true ? [...current, record.id] : current.filter((id) => id !== record.id))} aria-label={`#${record.id}`} />
            <div className='min-w-0 flex-1 space-y-2'>
              <div className='flex flex-wrap gap-x-4 gap-y-1 text-xs'>
                <span>#{record.id} · {t('User ID')} {record.user_id}</span><span>{record.model}</span><span>{t(record.status)}</span><span>{new Date(record.created_at).toLocaleString()}</span>
              </div>
              <p className='max-w-3xl break-words text-sm'>{record.prompt || t('Expired image')}</p>
              <div className='flex flex-wrap gap-2'>
                {record.assets.filter((asset) => asset.kind === 'result').map((asset) => <div key={asset.id} className='w-28'>
                  <GeneratedImage recordId={record.id} assetId={asset.id} admin />
                  <Button variant='ghost' size='icon-xs' title={t('Preview')} aria-label={t('Preview')} onClick={() => setPreviewAsset({ recordId: record.id, assetId: asset.id })}><Eye /></Button>
                  <Button variant='ghost' size='icon-xs' title={t('Download')} aria-label={t('Download')} onClick={() => void studioDownload(record.id, asset.id, true)}><Download /></Button>
                </div>)}
              </div>
            </div>
            <Button variant='ghost' size='icon-sm' title={t('Delete record')} aria-label={t('Delete record')} onClick={() => { setSelected([record.id]); setConfirmDelete(true) }}><Trash2 /></Button>
          </div>)}
          {!records.data?.data.length && <p className='text-muted-foreground p-5 text-sm'>{t('No generations yet')}</p>}
        </div>
        {records.data && records.data.total > 20 && <div className='flex items-center gap-2'>
          <Button variant='outline' size='sm' disabled={page === 1} onClick={() => setPage(page - 1)}>‹</Button>
          {page}
          <Button variant='outline' size='sm' disabled={page * 20 >= records.data.total} onClick={() => setPage(page + 1)}>›</Button>
        </div>}
      </>}
      <Dialog open={previewAsset !== null} onOpenChange={(open) => { if (!open) setPreviewAsset(null) }}>
        <DialogContent className='max-h-[90vh] overflow-y-auto sm:max-w-4xl'>
          <DialogHeader><DialogTitle>{t('Generated images')}</DialogTitle></DialogHeader>
          {previewAsset && <GeneratedImage recordId={previewAsset.recordId} assetId={previewAsset.assetId} admin />}
        </DialogContent>
      </Dialog>
      <ConfirmDialog open={confirmDelete} onOpenChange={setConfirmDelete} title={t('Delete records')} desc={t('Delete selected generations and their images?')} destructive handleConfirm={() => void batch('delete')} />
    </div>
  )
}
