import { Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { getUserAvatarFallback, getUserAvatarStyle } from '@/lib/avatar'

import { formatTokens } from '../lib/format'
import type { UserRanking } from '../types'

type UserLeaderboardSectionProps = {
  rows: UserRanking[]
}

export function UserLeaderboardSection(props: UserLeaderboardSectionProps) {
  const { t } = useTranslation()
  const left = props.rows.slice(0, 5)
  const right = props.rows.slice(5, 10)

  return (
    <section className='bg-card overflow-hidden rounded-lg border'>
      <header className='border-b px-5 py-4'>
        <h2 className='text-foreground inline-flex items-center gap-2 text-base font-semibold'>
          <Users className='text-primary size-4' aria-hidden='true' />
          {t('User Leaderboard')}
        </h2>
        <p className='text-muted-foreground mt-1 text-sm'>
          {t('Top users ranked by token usage for the selected period')}
        </p>
      </header>

      {props.rows.length === 0 ? (
        <div className='text-muted-foreground/80 px-5 py-10 text-center text-sm'>
          {t('No user usage data available')}
        </div>
      ) : (
        <div className='grid grid-cols-1 gap-x-8 px-5 py-3 md:grid-cols-2'>
          <UserList rows={left} />
          {right.length > 0 && <UserList rows={right} />}
        </div>
      )}
    </section>
  )
}

function UserList(props: { rows: UserRanking[] }) {
  return (
    <ol className='divide-border/60 divide-y'>
      {props.rows.map((row) => {
        const avatarName = row.display_name || row.username
        return (
          <li key={row.user_id} className='flex items-center gap-3 py-3'>
            <span className='text-muted-foreground/80 w-6 shrink-0 text-right font-mono text-xs tabular-nums'>
              {row.rank}.
            </span>
            <Avatar className='size-8 shrink-0'>
              <AvatarFallback style={getUserAvatarStyle(avatarName)}>
                {getUserAvatarFallback(avatarName)}
              </AvatarFallback>
            </Avatar>
            <div className='min-w-0 flex-1'>
              <div className='flex min-w-0 items-center gap-1.5'>
                <span className='text-foreground truncate text-sm font-medium'>
                  {row.username}
                </span>
                <span className='text-muted-foreground/70 shrink-0 font-mono text-[10px] tabular-nums'>
                  #{row.user_id}
                </span>
              </div>
              {row.display_name && row.display_name !== row.username && (
                <p className='text-muted-foreground/80 truncate text-xs'>
                  {row.display_name}
                </p>
              )}
            </div>
            <span className='text-foreground shrink-0 font-mono text-sm font-semibold tabular-nums'>
              {formatTokens(row.total_tokens)}
            </span>
          </li>
        )
      })}
    </ol>
  )
}
