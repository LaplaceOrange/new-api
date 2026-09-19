import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test, vi } from 'vitest'

import { parseRedeemResult, useRedemption } from '../use-redemption'

const toastSuccess = vi.fn()
const toastError = vi.fn()

vi.mock('i18next', () => ({
  default: {
    t: (key: string, options?: Record<string, unknown>) => {
      if (options?.title) return `subscription:${String(options.title)}`
      if (options?.quota) return `added:${String(options.quota)}`
      return key
    },
  },
}))

vi.mock('sonner', () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}))

vi.mock('@/lib/api', () => ({
  getSelf: vi.fn(async () => ({ success: true, data: {} })),
}))

vi.mock('@/lib/handle-server-error', () => ({
  handleServerError: vi.fn(),
}))

vi.mock('@/lib/format', () => ({
  formatQuota: (quota: number) => `quota:${quota}`,
}))

const redeemTopupCode = vi.fn()

vi.mock('../../api', () => ({
  redeemTopupCode: (...args: unknown[]) => redeemTopupCode(...args),
}))

afterEach(() => {
  toastSuccess.mockReset()
  toastError.mockReset()
  redeemTopupCode.mockReset()
})

describe('parseRedeemResult', () => {
  test('accepts a legacy quota number', () => {
    expect(parseRedeemResult(500)).toEqual({ quota: 500, plan_id: 0 })
  })

  test('reads quota and subscription fields from an object', () => {
    expect(
      parseRedeemResult({ quota: 0, plan_id: 9, plan_title: 'Pro' })
    ).toEqual({
      quota: 0,
      plan_id: 9,
      plan_title: 'Pro',
    })
  })
})

describe('useRedemption', () => {
  test('shows the quota toast when a quota code is redeemed', async () => {
    redeemTopupCode.mockResolvedValue({
      success: true,
      data: { quota: 500000, plan_id: 0 },
    })
    const { result } = renderHook(() => useRedemption())

    await act(async () => {
      await expect(result.current.redeemCode('CODE')).resolves.toBe(true)
    })

    await waitFor(() => expect(toastSuccess).toHaveBeenCalled())
    expect(toastSuccess.mock.calls[0]?.[0]).toBe('added:quota:500000')
  })

  test('shows the subscription toast when a plan is granted', async () => {
    redeemTopupCode.mockResolvedValue({
      success: true,
      data: { quota: 0, plan_id: 3, plan_title: 'Gift Plan' },
    })
    const { result } = renderHook(() => useRedemption())

    await act(async () => {
      await expect(result.current.redeemCode('CODE')).resolves.toBe(true)
    })

    await waitFor(() => expect(toastSuccess).toHaveBeenCalled())
    expect(toastSuccess.mock.calls[0]?.[0]).toBe('subscription:Gift Plan')
  })
})
