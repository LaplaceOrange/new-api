import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Download, ImagePlus, Play, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
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
import { handleServerError } from '@/lib/handle-server-error'
import { api } from '@/lib/api'

import { studioAsset, studioDownload, type StudioLegal, type StudioOptions, type StudioRecord } from './api'

const ratios = [
  { label: '1:1', w: 1, h: 1 },
  { label: '3:2', w: 3, h: 2 },
  { label: '2:3', w: 2, h: 3 },
  { label: '4:3', w: 4, h: 3 },
  { label: '3:4', w: 3, h: 4 },
  { label: '16:9', w: 16, h: 9 },
  { label: '9:16', w: 9, h: 16 },
] as const

const tabLabels = {
  history: 'Generation history',
  create: 'Create',
  gallery: 'Generated images',
}

export function GeneratedImage(props: { recordId: number; assetId: number; admin?: boolean }) {
  const [url, setUrl] = useState('')
  useEffect(() => {
    let active = true
    studioAsset(props.recordId, props.assetId, props.admin)
      .then((value) => { if (active) setUrl(value); else URL.revokeObjectURL(value) })
      .catch(() => { if (active) setUrl('') })
    return () => { active = false }
  }, [props.recordId, props.assetId, props.admin])
  useEffect(() => () => { if (url) URL.revokeObjectURL(url) }, [url])
  if (!url) return <div className='bg-muted aspect-square animate-pulse' />
  return <img src={url} alt='' className='max-h-[65vh] w-full object-contain' />
}

export function ImageStudio() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<'history' | 'create' | 'gallery'>('create')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<number | null>(null)
  const [prompt, setPrompt] = useState('')
  const [modelName, setModelName] = useState('')
  const [group, setGroup] = useState('')
  const [ratio, setRatio] = useState('auto')
  const [variant, setVariant] = useState(1024)
  const [width, setWidth] = useState('')
  const [height, setHeight] = useState('')
  const [count, setCount] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const [fileInputVersion, setFileInputVersion] = useState(0)
  const [accepted, setAccepted] = useState(false)
  const [legalKind, setLegalKind] = useState<'agreement' | 'privacy' | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const options = useQuery({
    queryKey: ['image-studio', 'options'],
    queryFn: async () => (await api.get<StudioOptions>('/api/image-studio/options')).data,
  })
  const records = useQuery({
    queryKey: ['image-studio', 'records', page],
    queryFn: async () => (await api.get<{ data: StudioRecord[]; total: number }>(`/api/image-studio/records?page=${page}`)).data,
    refetchInterval: (query) => query.state.data?.data.some((row) => row.status === 'processing') ? 3000 : false,
  })
  const legal = useQuery({
    queryKey: ['image-studio', 'legal', legalKind],
    queryFn: async () => (await api.get<StudioLegal>(`/api/image-studio/legal/${legalKind}`)).data,
    enabled: legalKind !== null,
  })
  const model = options.data?.models.find((item) => item.name === modelName) ?? options.data?.models[0]
  const groups = useMemo(() => Object.entries(model?.groups ?? {}), [model])
  const activeGroup = groups.some(([name]) => name === group) ? group : groups[0]?.[0] ?? ''
  const price = model && activeGroup ? model.price * model.groups[activeGroup] * (Number(count) || 1) : 0
  const list = records.data?.data ?? []
  const record = list.find((item) => item.id === selected) ?? list[0]

  useEffect(() => { setAccepted(false) }, [options.data?.agreement_revision, options.data?.privacy_revision])

  function selectRatio(label: string, w?: number, h?: number, edge = variant) {
    setRatio(label)
    setVariant(edge)
    if (!w || !h) { setWidth(''); setHeight(''); return }
    setWidth(String(Math.round(edge * w / Math.max(w, h))))
    setHeight(String(Math.round(edge * h / Math.max(w, h))))
  }

  async function generate() {
    if (!model || !activeGroup || !options.data || !accepted) return
    setSubmitting(true)
    const form = new FormData()
    form.set('model', model.name)
    form.set('group', activeGroup)
    form.set('prompt', prompt)
    form.set('n', count || '1')
    if (width && height) form.set('size', `${width}x${height}`)
    form.set('consent', 'true')
    form.set('agreement_revision', options.data.agreement_revision)
    form.set('privacy_revision', options.data.privacy_revision)
    files.forEach((file) => form.append('image', file))
    try {
      const response = await api.post<{ id: number }>('/api/image-studio/generate', form)
      setSelected(response.data.id)
      setTab('gallery')
      setAccepted(false)
      setFiles([])
      setFileInputVersion((value) => value + 1)
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'records'] })
      toast.success(t('Images generated'))
    } catch (error) {
      handleServerError(error, t('Image generation failed'))
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'records'] })
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'options'] })
    } finally { setSubmitting(false); setAccepted(false) }
  }

  async function deleteRecord() {
    if (!record) return
    try {
      await api.delete(`/api/image-studio/records/${record.id}`)
      setSelected(null)
      await queryClient.invalidateQueries({ queryKey: ['image-studio', 'records'] })
      setDeleting(false)
    } catch { toast.error(t('Failed to delete record')) }
  }

  return (
    <div className='flex h-full min-h-[calc(100vh-6rem)] flex-col bg-background'>
      <div className='border-b px-5 py-4'>
        <h1 className='text-xl font-semibold'>{t('Online Image Studio')}</h1>
      </div>
      <div className='flex gap-1 border-b px-3 py-2 lg:hidden'>
        {(['history', 'create', 'gallery'] as const).map((value) => (
          <Button key={value} size='sm' variant={tab === value ? 'secondary' : 'ghost'} onClick={() => setTab(value)}>
            {t(tabLabels[value])}
          </Button>
        ))}
      </div>
      <div className='grid flex-1 min-h-0 lg:grid-cols-[260px_minmax(340px,400px)_minmax(0,1fr)]'>
        <aside className={`border-r p-4 ${tab !== 'history' ? 'hidden lg:block' : ''}`}>
          <h2 className='mb-4 text-sm font-semibold'>{t('Generation history')}</h2>
          {list.length === 0 && <p className='text-muted-foreground text-sm'>{t('No generations yet')}</p>}
          <div className='space-y-1'>
            {list.map((item) => (
              <Button key={item.id} variant={record?.id === item.id ? 'secondary' : 'ghost'} className='h-auto w-full justify-start whitespace-normal px-3 py-3 text-left' onClick={() => { setSelected(item.id); setTab('gallery') }}>
                <span className='min-w-0'>
                  <span className='block truncate font-medium'>{item.prompt || t('Expired image')}</span>
                  <span className='text-muted-foreground block text-xs'>{item.model} · {t(item.status)} · {new Date(item.created_at).toLocaleString()}</span>
                </span>
              </Button>
            ))}
          </div>
          {records.data && records.data.total > 20 && (
            <div className='mt-4 flex items-center gap-2'>
              <Button variant='outline' size='sm' disabled={page === 1} onClick={() => setPage(page - 1)}>‹</Button>
              <span className='text-sm'>{page}</span>
              <Button variant='outline' size='sm' disabled={page * 20 >= records.data.total} onClick={() => setPage(page + 1)}>›</Button>
            </div>
          )}
        </aside>
        <section className={`overflow-y-auto border-r p-5 ${tab !== 'create' ? 'hidden lg:block' : ''}`}>
          <h2 className='mb-5 text-sm font-semibold'>{t('Create image')}</h2>
          {!options.data?.ready && <p className='text-muted-foreground mb-4 text-sm'>{t('Image Studio is not configured yet')}</p>}
          <div className='space-y-5'>
            <label className='block text-sm font-medium'>{t('Prompt')}<Textarea className='mt-2 min-h-32' value={prompt} onChange={(event) => setPrompt(event.target.value)} maxLength={10000} /></label>
            <label className='block text-sm font-medium'>
              {t('Reference images')} <span className='text-muted-foreground font-normal'>({t('Optional')})</span>
              <div className='mt-2 flex items-center gap-2'>
                <Input key={`${model?.name}-${fileInputVersion}`} type='file' accept='image/png,image/jpeg,image/webp' multiple disabled={!model?.allow_edits} onChange={(event) => setFiles([...(event.target.files ?? [])])} />
                <ImagePlus className='text-muted-foreground size-4 shrink-0' />
              </div>
              {!model?.allow_edits && <span className='text-muted-foreground mt-1 block text-xs'>{t('This model does not support reference images')}</span>}
              {files.length > 0 && <span className='text-muted-foreground mt-1 block text-xs'>{files.length} / 4</span>}
            </label>
            <div className='grid grid-cols-2 gap-3'>
              <label className='text-sm font-medium'>{t('Model')}
                <NativeSelect className='mt-2 w-full' value={model?.name ?? ''} onChange={(event) => { setModelName(event.target.value); setGroup(''); setFiles([]) }}>
                  {(options.data?.models ?? []).map((item) => <NativeSelectOption key={item.name} value={item.name}>{item.name}</NativeSelectOption>)}
                </NativeSelect>
              </label>
              <label className='text-sm font-medium'>{t('Group')}
                <NativeSelect className='mt-2 w-full' value={activeGroup} onChange={(event) => setGroup(event.target.value)}>
                  {groups.map(([name, multiplier]) => <NativeSelectOption key={name} value={name}>{name} · ×{multiplier}</NativeSelectOption>)}
                </NativeSelect>
              </label>
            </div>
            <p className='text-muted-foreground text-xs'>
              <Link to='/wallet' hash='subscriptions' className='text-primary underline underline-offset-2'>{t('Purchase any subscription')}</Link>{t(' to upgrade your user group and reduce the image generation multiplier.')}
            </p>
            <div>
              <p className='mb-2 text-sm font-medium'>{t('Aspect ratio')}</p>
              <div className='grid grid-cols-4 gap-2'>
                <Button variant={ratio === 'auto' ? 'secondary' : 'outline'} className='h-16' onClick={() => selectRatio('auto')}>{t('Auto')}</Button>
                {ratios.map((item) => (
                  <Button key={item.label} variant={ratio === item.label ? 'secondary' : 'outline'} className='h-16 flex-col gap-1' onClick={() => selectRatio(item.label, item.w, item.h)}>
                    <span className='border-foreground/60 block border' style={{ width: `${Math.round(30 * item.w / Math.max(item.w, item.h))}px`, height: `${Math.round(30 * item.h / Math.max(item.w, item.h))}px` }} />
                    <span className='text-xs'>{item.label}</span>
                  </Button>
                ))}
              </div>
              {ratio !== 'auto' && <div className='mt-2 flex gap-2'>
                {[1024, 2048, 4096].map((edge) => <Button key={edge} size='sm' variant={variant === edge ? 'secondary' : 'outline'} onClick={() => { const item = ratios.find((entry) => entry.label === ratio); if (item) selectRatio(item.label, item.w, item.h, edge) }}>{`${edge / 1024}K`}</Button>)}
              </div>}
            </div>
            <div className='grid grid-cols-2 gap-3'>
              <label className='text-sm font-medium'>{t('Width')}<Input className='mt-2' type='number' min='256' max='4096' value={width} onChange={(event) => { setWidth(event.target.value); setRatio('custom') }} /></label>
              <label className='text-sm font-medium'>{t('Height')}<Input className='mt-2' type='number' min='256' max='4096' value={height} onChange={(event) => { setHeight(event.target.value); setRatio('custom') }} /></label>
            </div>
            <label className='block text-sm font-medium'>{t('Number of images')}<Input className='mt-2' type='number' min='1' max='128' placeholder='1' value={count} onChange={(event) => setCount(event.target.value)} /></label>
            <div className='border-t pt-4'>
              <div className='mb-4 flex items-baseline justify-between text-sm'><span>{t('Estimated cost')}</span><strong className='text-lg'>${price.toFixed(4)}</strong></div>
              <div className='mb-4 flex items-start gap-2 text-xs'>
                <Checkbox checked={accepted} onCheckedChange={(value) => setAccepted(value === true)} aria-label={t('Accept image studio terms')} />
                <span>{t('I have read and agree to the')}{' '}
                  <Button variant='link' size='xs' className='h-auto p-0 text-xs' onClick={() => setLegalKind('agreement')}>{t('Image Studio User Agreement')}</Button>{' '}{t('and the')}{' '}
                  <Button variant='link' size='xs' className='h-auto p-0 text-xs' onClick={() => setLegalKind('privacy')}>{t('Image Studio Privacy Policy')}</Button>
                </span>
              </div>
              <Button className='w-full' disabled={!options.data?.ready || !prompt.trim() || !accepted || submitting || files.length > 4 || (files.length > 0 && !model?.allow_edits) || (!!width !== !!height) || (!!count && (!Number.isInteger(Number(count)) || Number(count) < 1 || Number(count) > 128))} onClick={generate}>
                <Play />{submitting ? t('Generating...') : t('Start generation')}
              </Button>
            </div>
          </div>
        </section>
        <main className={`min-w-0 overflow-y-auto p-5 ${tab !== 'gallery' ? 'hidden lg:block' : ''}`}>
          <div className='mb-5 flex items-center justify-between'>
            <h2 className='text-sm font-semibold'>{t('Generated images')}</h2>
            {record && <Button variant='ghost' size='icon-sm' disabled={record.status === 'processing'} title={t('Delete record')} aria-label={t('Delete record')} onClick={() => setDeleting(true)}><Trash2 /></Button>}
          </div>
          {!record && <div className='text-muted-foreground flex min-h-72 items-center justify-center text-sm'>{t('Your images will appear here')}</div>}
          {record && <>
            <div className='mb-4 space-y-1 text-sm'>
              <p className='font-medium break-words'>{record.prompt || t('Expired image')}</p>
              <p className='text-muted-foreground'>{record.model} · {record.size || t('Auto')} · {t(record.status)}</p>
              {record.actual_price != null && <p className='text-muted-foreground'>{t('Actual cost')}: ${record.actual_price.toFixed(4)}</p>}
              {record.error && <p className='text-destructive break-words'>{record.error}</p>}
              <p className='text-muted-foreground text-xs'>{t('Expires')}: {new Date(record.expires_at).toLocaleString()}</p>
            </div>
            {record.assets.some((asset) => asset.kind === 'reference') && <div className='mb-4'>
              <h3 className='text-muted-foreground mb-2 text-xs font-medium'>{t('Reference images')}</h3>
              <div className='flex flex-wrap gap-2'>
                {record.assets.filter((asset) => asset.kind === 'reference').map((asset) => (
                  <div key={asset.id} className='size-20 overflow-hidden border'>
                    <GeneratedImage recordId={record.id} assetId={asset.id} />
                  </div>
                ))}
              </div>
            </div>}
            <div className='grid gap-4 xl:grid-cols-2'>
              {record.assets.filter((asset) => asset.kind === 'result').map((asset) => (
                <div key={asset.id} className='border p-2'>
                  <GeneratedImage recordId={record.id} assetId={asset.id} />
                  <Button variant='ghost' size='sm' className='mt-2' onClick={() => void studioDownload(record.id, asset.id)}><Download />{t('Download')}</Button>
                </div>
              ))}
            </div>
          </>}
        </main>
      </div>
      <Dialog open={legalKind !== null} onOpenChange={(open) => { if (!open) setLegalKind(null) }}>
        <DialogContent className='max-h-[85vh] overflow-y-auto sm:max-w-2xl'>
          <DialogHeader><DialogTitle>{t(legalKind === 'agreement' ? 'Image Studio User Agreement' : 'Image Studio Privacy Policy')}</DialogTitle></DialogHeader>
          {legal.data && <>
            <p className='text-muted-foreground text-xs'>{legal.data.operator_name} · {legal.data.contact_email}</p>
            <RichContent mode='markdown' content={legal.data.content} />
          </>}
        </DialogContent>
      </Dialog>
      <ConfirmDialog open={deleting} onOpenChange={setDeleting} title={t('Delete record')} desc={t('Delete this generation and its images?')} destructive handleConfirm={() => void deleteRecord()} />
    </div>
  )
}
