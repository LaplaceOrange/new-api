import type { ColumnFiltersState } from '@tanstack/react-table'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { expect, it } from 'vitest'

import { DataTableToolbar } from '../../toolbar/toolbar'
import { useDataTable } from '../use-data-table'

const rows = [{ name: 'Production' }, { name: 'Backup' }]
const columns = [{ accessorKey: 'name' }]

function TableWithToolbar(props: { controlled?: boolean }) {
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const { table } = useDataTable({
    data: rows,
    columns,
    columnFilters: props.controlled ? columnFilters : undefined,
    onColumnFiltersChange: props.controlled ? setColumnFilters : undefined,
  })

  return (
    <>
      <DataTableToolbar table={table} searchKey='name' />
      <ul>
        {table.getRowModel().rows.map((row) => (
          <li key={row.id}>{row.original.name}</li>
        ))}
      </ul>
    </>
  )
}

it('renders the toolbar when no column filters are configured', () => {
  render(<TableWithToolbar />)

  expect(screen.getByRole('textbox')).toBeVisible()
})

it.each([false, true])(
  'filters and resets rows with controlled column filters set to %s',
  async (controlled) => {
    const user = userEvent.setup()
    render(<TableWithToolbar controlled={controlled} />)

    await user.type(screen.getByRole('textbox'), 'Production')
    expect(screen.getByText('Production')).toBeVisible()
    expect(screen.queryByText('Backup')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(screen.getByRole('textbox')).toHaveValue('')
    expect(screen.getByText('Backup')).toBeVisible()
  }
)
