/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronLeft, ChevronRight, Search, Unlock } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  getSensitiveWordIPBans,
  getSensitiveWordUserBans,
  unbanSensitiveWordIP,
  unbanSensitiveWordUser,
} from '../api'
import type { SensitiveWordBanPage } from '../types'

function formatTimestamp(timestamp: number): string {
  if (!timestamp) return '-'
  return new Date(timestamp * 1000).toLocaleString()
}

type BanTableProps = {
  kind: 'user' | 'ip'
}

type SensitiveWordBanRow = {
  id: number
  value: string | number
  label: string
  hit_count: number
  first_hit_at: number
  last_hit_at: number
  banned_at: number
}

function SensitiveWordBanTable(props: BanTableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const queryKey = ['sensitive-word-bans', props.kind, page, search]
  const query = useQuery({
    queryKey,
    queryFn: async (): Promise<SensitiveWordBanPage<SensitiveWordBanRow>> => {
      if (props.kind === 'user') {
        const data = await getSensitiveWordUserBans(page, search)
        return {
          ...data,
          items: data.items.map((item) => ({
            ...item,
            value: item.user_id,
            label: `${item.username || '-'} (#${item.user_id})`,
          })),
        }
      }
      const data = await getSensitiveWordIPBans(page, search)
      return {
        ...data,
        items: data.items.map((item) => ({
          ...item,
          value: item.ip,
          label: item.ip,
        })),
      }
    },
  })
  const unban = useMutation({
    mutationFn: (value: string | number) =>
      props.kind === 'user'
        ? unbanSensitiveWordUser(Number(value))
        : unbanSensitiveWordIP(String(value)),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ['sensitive-word-bans'] }),
  })
  const totalPages = Math.max(1, Math.ceil((query.data?.total ?? 0) / 10))
  const submitSearch = () => {
    setPage(1)
    setSearch(searchInput.trim())
  }

  return (
    <div className='space-y-3'>
      <div className='flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
        <h3 className='text-sm font-medium'>
          {props.kind === 'user'
            ? t('Sensitive-word banned users')
            : t('Sensitive-word banned IPs')}
        </h3>
        <div
          className='flex w-full gap-2 sm:w-auto'
        >
          <Input
            value={searchInput}
            onChange={(event) => setSearchInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                submitSearch()
              }
            }}
            placeholder={
              props.kind === 'user'
                ? t('Search user ID or username')
                : t('Search IP address')
            }
          />
          <Button type='button' size='icon' aria-label={t('Search')} onClick={submitSearch}>
            <Search />
          </Button>
        </div>
      </div>
      <div className='rounded-md border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>
                {props.kind === 'user' ? t('User') : t('IP address')}
              </TableHead>
              <TableHead>{t('Hit count')}</TableHead>
              <TableHead>{t('First hit')}</TableHead>
              <TableHead>{t('Last hit')}</TableHead>
              <TableHead>{t('Banned at')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {query.data?.items.map((item) => {
              return (
                <TableRow key={`${props.kind}-${item.id}`}>
                  <TableCell>{item.label}</TableCell>
                  <TableCell>{item.hit_count}</TableCell>
                  <TableCell>{formatTimestamp(item.first_hit_at)}</TableCell>
                  <TableCell>{formatTimestamp(item.last_hit_at)}</TableCell>
                  <TableCell>{formatTimestamp(item.banned_at)}</TableCell>
                  <TableCell className='text-right'>
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      disabled={unban.isPending}
                      onClick={() => unban.mutate(item.value)}
                    >
                      <Unlock />
                      {t('Unban')}
                    </Button>
                  </TableCell>
                </TableRow>
              )
            })}
            {!query.isLoading && (query.data?.items.length ?? 0) === 0 && (
              <TableRow>
                <TableCell colSpan={6} className='text-muted-foreground h-24 text-center'>
                  {t('No banned records')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className='flex items-center justify-end gap-2'>
        <span className='text-muted-foreground text-sm'>
          {t('Page {{page}} of {{total}}', { page, total: totalPages })}
        </span>
        <Button
          type='button'
          size='icon-sm'
          variant='outline'
          aria-label={t('Previous page')}
          disabled={page <= 1}
          onClick={() => setPage((value) => Math.max(1, value - 1))}
        >
          <ChevronLeft />
        </Button>
        <Button
          type='button'
          size='icon-sm'
          variant='outline'
          aria-label={t('Next page')}
          disabled={page >= totalPages}
          onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  )
}

export function SensitiveWordBanLists() {
  return (
    <div className='space-y-8'>
      <SensitiveWordBanTable kind='user' />
      <SensitiveWordBanTable kind='ip' />
    </div>
  )
}
