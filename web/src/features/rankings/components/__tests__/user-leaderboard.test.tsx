import { render, screen } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, describe, expect, test } from 'vitest'

import { UserLeaderboardSection } from '../user-leaderboard'

const rows = Array.from({ length: 10 }, (_, index) => ({
  rank: index + 1,
  user_id: index + 101,
  username: `user-${index + 1}`,
  total_tokens: (10 - index) * 1000,
}))

describe('rankings user leaderboard', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'User Leaderboard': 'User Leaderboard',
      'Top users ranked by token usage for the selected period':
        'Top users ranked by token usage for the selected period',
      'No user usage data available': 'No user usage data available',
    })
  })

  test('renders the top ten users in two five-user columns', () => {
    render(<UserLeaderboardSection rows={rows} />)

    const section = screen
      .getByRole('heading', { name: 'User Leaderboard' })
      .closest('section')
    expect(section).not.toBeNull()
    const lists = section?.querySelectorAll('ol')
    expect(lists).toHaveLength(2)
    expect(lists?.[0].querySelectorAll('li')).toHaveLength(5)
    expect(lists?.[1].querySelectorAll('li')).toHaveLength(5)
    expect(screen.getByText('user-1')).toBeInTheDocument()
    expect(screen.getByText('user-10')).toBeInTheDocument()
    expect(screen.getByText('#101')).toBeInTheDocument()
    expect(screen.getByText('#110')).toBeInTheDocument()
  })

  test('shows the empty state when no user usage exists', () => {
    render(<UserLeaderboardSection rows={[]} />)

    expect(screen.getByText('No user usage data available')).toBeInTheDocument()
    expect(screen.queryAllByRole('list')).toHaveLength(0)
  })
})
