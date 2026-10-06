import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { afterEach, expect, it } from 'vitest'

import {
  upstreamEditorDefaults,
  upstreamEditorPayload,
  upstreamEditorSchema,
  type UpstreamEditorValues,
} from '../../lib/editor'
import type { Upstream, UpstreamPayload } from '../../types'
import { UpstreamCredentialsFields } from '../upstream-credentials-fields'

afterEach(cleanup)

function Harness(props: {
  busy?: boolean
  upstream?: Upstream
  onSubmit?: (payload: UpstreamPayload) => void
}) {
  const form = useForm<UpstreamEditorValues>({
    defaultValues: upstreamEditorDefaults(props.upstream ?? null),
  })
  return (
    <form
      onSubmit={form.handleSubmit((values) =>
        props.onSubmit?.(upstreamEditorPayload(values))
      )}
    >
      <UpstreamCredentialsFields
        form={form}
        formId='credentials'
        busy={props.busy ?? false}
        upstream={props.upstream ?? null}
      />
      <button type='submit'>Submit</button>
    </form>
  )
}

it('switches to account login without exposing or submitting the JWT fields', async () => {
  const user = userEvent.setup()
  let result: UpstreamPayload | undefined
  render(
    <Harness
      onSubmit={(payload) => {
        result = payload
      }}
    />
  )
  await user.type(screen.getByLabelText('Access token'), 'discard-this-jwt')
  await user.click(screen.getByRole('button', { name: 'Account password' }))
  expect(screen.queryByLabelText('Access token')).not.toBeInTheDocument()
  expect(
    screen.queryByLabelText('Automatically renew JWT')
  ).not.toBeInTheDocument()
  await user.type(screen.getByLabelText('Account email'), 'account@example.com')
  await user.type(
    screen.getByLabelText('Account password'),
    '  exact password  '
  )
  expect(screen.getByLabelText('Account password')).toHaveAttribute(
    'type',
    'password'
  )
  await user.click(screen.getByRole('button', { name: 'Submit' }))
  expect(result).toMatchObject({
    auth_mode: 'password',
    account_email: 'account@example.com',
    account_password: '  exact password  ',
    auto_refresh_token: false,
  })
  expect(result).not.toHaveProperty('access_token')
  expect(result).not.toHaveProperty('refresh_token')
})

it('keeps saved account credentials private and omits blank replacements', () => {
  const upstream = {
    auth_mode: 'password',
    has_account_credentials: true,
    addresses: ['https://upstream.example'],
  } as Upstream
  render(<Harness upstream={upstream} />)
  expect(screen.getByLabelText('Account email')).toHaveValue('')
  expect(screen.getByLabelText('Account password')).toHaveValue('')
  const payload = upstreamEditorPayload(upstreamEditorDefaults(upstream))
  expect(payload).not.toHaveProperty('account_email')
  expect(payload).not.toHaveProperty('account_password')
})

it('disables authentication selection and credential editing while a save is pending', () => {
  render(<Harness busy />)
  expect(
    screen.getByRole('button', { name: 'Account password' })
  ).toBeDisabled()
  expect(screen.getByRole('button', { name: 'JWT' })).toBeDisabled()
  expect(screen.getByLabelText('Access token')).toBeDisabled()
})

it('supports keyboard selection and clears hidden secrets when changing modes', async () => {
  const user = userEvent.setup()
  render(<Harness />)
  const account = screen.getByRole('button', { name: 'Account password' })
  account.focus()
  await user.keyboard('{Enter}')
  expect(account).toHaveAttribute('aria-pressed', 'true')
  await user.type(screen.getByLabelText('Account password'), 'do-not-retain')
  await user.click(screen.getByRole('button', { name: 'JWT' }))
  await user.click(screen.getByRole('button', { name: 'Account password' }))
  expect(screen.getByLabelText('Account password')).toHaveValue('')
})

it('rejects malformed account emails while allowing existing credentials to remain blank', () => {
  const values = {
    ...upstreamEditorDefaults(null),
    name: 'upstream',
    addresses: 'https://upstream.example',
    primary_url: 'https://upstream.example',
    auth_mode: 'password' as const,
  }
  expect(upstreamEditorSchema.safeParse(values).success).toBe(true)
  const result = upstreamEditorSchema.safeParse({
    ...values,
    account_email: 'not-an-email',
  })
  expect(result.success).toBe(false)
  if (!result.success) {
    expect(result.error.issues).toContainEqual(
      expect.objectContaining({
        path: ['account_email'],
        message: 'Enter a valid account email',
      })
    )
  }
})
