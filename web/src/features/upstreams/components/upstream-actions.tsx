import { Pencil, RefreshCw, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { Upstream } from '../types'

export function UpstreamActions(props: {
  upstream: Upstream
  root: boolean
  pending: boolean
  onRefresh: (id: number) => void
  onEdit: (upstream: Upstream) => void
  onDelete: (upstream: Upstream) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='flex flex-wrap gap-1'>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon'
              aria-label={t('Refresh upstream')}
              disabled={props.pending || props.upstream.refreshing}
              onClick={() => props.onRefresh(props.upstream.id)}
            />
          }
        >
          <RefreshCw
            className={
              props.upstream.refreshing ? 'size-4 animate-spin' : 'size-4'
            }
          />
        </TooltipTrigger>
        <TooltipContent>{t('Refresh upstream')}</TooltipContent>
      </Tooltip>
      {props.root && (
        <>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon'
                  aria-label={t('Edit upstream')}
                  onClick={() => props.onEdit(props.upstream)}
                />
              }
            >
              <Pencil className='size-4' />
            </TooltipTrigger>
            <TooltipContent>{t('Edit upstream')}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant='ghost'
                  size='icon'
                  aria-label={t('Delete upstream')}
                  onClick={() => props.onDelete(props.upstream)}
                />
              }
            >
              <Trash2 className='text-destructive size-4' />
            </TooltipTrigger>
            <TooltipContent>{t('Delete upstream')}</TooltipContent>
          </Tooltip>
        </>
      )}
    </div>
  )
}
