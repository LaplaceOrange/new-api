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
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'

import { getLobeIcon, getLobeIconNames } from '../lobe-icon'

it.each(['OpenAI', 'DeepSeek.Color', 'Claude'])(
  'renders %s with the requested size',
  async (name) => {
    const view = render(<>{getLobeIcon(name, 28)}</>)
    await screen.findByTitle(name.split('.')[0])
    expect(view.container.querySelector('svg')).toHaveAttribute('width', '28')
  }
)

it('preserves explicit size and the base icon for an unavailable variant', async () => {
  const view = render(<>{getLobeIcon('OpenAI.Color.size={32}', 20)}</>)
  await screen.findByTitle('OpenAI')
  expect(view.container.querySelector('svg')).toHaveAttribute('width', '32')
})

it('keeps placeholders and custom icons available', () => {
  const view = render(
    <>
      {getLobeIcon(null)}
      {getLobeIcon('UnknownIcon')}
      {getLobeIcon('Sub2API')}
      {getLobeIcon('WorkBuddy')}
    </>
  )
  expect(screen.getByText('?')).toBeVisible()
  expect(screen.getByText('U')).toBeVisible()
  expect(view.container.querySelector('svg')).toBeInTheDocument()
  expect(view.container.querySelector('img')).toHaveAttribute('width', '20')
})

it('offers the installed icon catalog and custom names in the selector', () => {
  const names = getLobeIconNames()
  expect(names).toEqual(
    expect.arrayContaining([
      'OpenAI',
      'DeepSeek.Color',
      'Claude',
      'XiaomiMiMo',
      'WorkBuddy',
      'Sub2API',
    ])
  )
  expect(new Set(names).size).toBe(names.length)
})
