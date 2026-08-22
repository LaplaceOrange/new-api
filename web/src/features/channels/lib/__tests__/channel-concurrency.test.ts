import { describe, expect, test } from 'vitest'

import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from '../channel-form'
import {
  aggregateChannelsByTag,
  formatChannelConcurrency,
} from '../channel-utils'
import { channelSchema, type Channel } from '../../types'

function createChannel(overrides: Partial<Channel> = {}): Channel {
  return channelSchema.parse({
    id: 1,
    type: 1,
    key: 'key',
    status: 1,
    name: 'channel',
    created_time: 0,
    test_time: 0,
    response_time: 100,
    balance_updated_time: 0,
    tag: 'group-a',
    ...overrides,
  })
}

describe('channel concurrency form and display', () => {
  test('accepts blank, zero, and positive integer limits only', () => {
    const validDefaults = {
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: 'channel',
      key: 'key',
      models: 'gpt-4',
    }
    for (const value of [undefined, 0, 8]) {
      expect(
        channelFormSchema.safeParse({
          ...validDefaults,
          concurrency_limit: value,
        }).success
      ).toBe(true)
    }
    expect(
      channelFormSchema.safeParse({
        ...validDefaults,
        concurrency_limit: -1,
      }).success
    ).toBe(false)
    expect(
      channelFormSchema.safeParse({
        ...validDefaults,
        concurrency_limit: 1.5,
      }).success
    ).toBe(false)
  })

  test('normalizes blank and zero limits to null in create and update payloads', () => {
    const blank = { ...CHANNEL_FORM_DEFAULT_VALUES, concurrency_limit: undefined }
    const zero = { ...CHANNEL_FORM_DEFAULT_VALUES, concurrency_limit: 0 }
    const positive = { ...CHANNEL_FORM_DEFAULT_VALUES, concurrency_limit: 8 }

    expect(transformFormDataToCreatePayload(blank).channel.concurrency_limit).toBeNull()
    expect(transformFormDataToCreatePayload(zero).channel.concurrency_limit).toBeNull()
    expect(transformFormDataToCreatePayload(positive).channel.concurrency_limit).toBe(8)
    expect(transformFormDataToUpdatePayload(blank, 1).concurrency_limit).toBeNull()
    expect(transformFormDataToUpdatePayload(zero, 1).concurrency_limit).toBeNull()
    expect(transformFormDataToUpdatePayload(positive, 1).concurrency_limit).toBe(8)
  })

  test('loads a persisted positive limit into form defaults', () => {
    const defaults = transformChannelToFormDefaults(
      createChannel({ concurrency_limit: 8 })
    )
    expect(defaults.concurrency_limit).toBe(8)
    expect(
      transformChannelToFormDefaults(createChannel({ concurrency_limit: null }))
        .concurrency_limit
    ).toBeUndefined()
  })

  test('formats known, unlimited, and unavailable concurrency states', () => {
    expect(
      formatChannelConcurrency(
        createChannel({
          concurrency_known: true,
          current_concurrency: 3,
          concurrency_limit: 10,
        })
      )
    ).toBe('3 / 10')
    expect(
      formatChannelConcurrency(
        createChannel({
          concurrency_known: true,
          current_concurrency: 3,
          concurrency_limit: null,
        })
      )
    ).toBe('3 / ∞')
    expect(
      formatChannelConcurrency(createChannel({ concurrency_known: false }))
    ).toBe('—')
  })
})

describe('channel tag concurrency aggregation', () => {
  test('sums current counts and finite limits', () => {
    const [row] = aggregateChannelsByTag([
      createChannel({ id: 1, current_concurrency: 2, concurrency_limit: 5, concurrency_known: true }),
      createChannel({ id: 2, current_concurrency: 3, concurrency_limit: 7, concurrency_known: true }),
    ])
    expect(row).toMatchObject({
      current_concurrency: 5,
      concurrency_limit: 12,
      concurrency_known: true,
    })
  })

  test('uses unlimited aggregate limit when any child is unlimited', () => {
    const [row] = aggregateChannelsByTag([
      createChannel({ id: 1, current_concurrency: 2, concurrency_limit: 5, concurrency_known: true }),
      createChannel({ id: 2, current_concurrency: 3, concurrency_limit: null, concurrency_known: true }),
    ])
    expect(row).toMatchObject({
      current_concurrency: 5,
      concurrency_limit: null,
      concurrency_known: true,
    })
  })

  test('marks aggregate monitoring unavailable when any child is unknown', () => {
    const [row] = aggregateChannelsByTag([
      createChannel({ id: 1, current_concurrency: 2, concurrency_limit: 5, concurrency_known: true }),
      createChannel({ id: 2, concurrency_known: false }),
    ])
    expect(row).toMatchObject({
      concurrency_known: false,
    })
  })
})
