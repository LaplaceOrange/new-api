import { describe, expect, it } from 'vitest'

import {
  getFirstResponseTimeColor,
  getResponseTimeColor,
  getSensitiveWordMatches,
  getThroughputColor,
  parseLogOther,
} from '../format'

describe('sensitive-word usage log data', () => {
  it('returns trimmed matched words from structured log data', () => {
    const other = parseLogOther(
      JSON.stringify({
        sensitive_words: [' xxx ', '', 123, 'yyy'],
      })
    )

    expect(getSensitiveWordMatches(other)).toEqual(['xxx', 'yyy'])
  })

  it('returns no matches for missing or malformed structured data', () => {
    expect(getSensitiveWordMatches(null)).toEqual([])
    expect(
      getSensitiveWordMatches(parseLogOther('{"sensitive_words":"xxx"}'))
    ).toEqual([])
  })
})

describe('request performance colors', () => {
  it.each([
    [0, 'success'],
    [9.999, 'success'],
    [10, 'warning'],
    [30, 'orange'],
    [60, 'danger'],
    [-1, 'neutral'],
    [Number.NaN, 'neutral'],
  ])('classifies first-token latency %s as %s', (seconds, expected) => {
    expect(getFirstResponseTimeColor(seconds)).toBe(expected)
  })

  it.each([
    [0, 'success'],
    [59.999, 'success'],
    [60, 'warning'],
    [180, 'orange'],
    [300, 'danger'],
    [Infinity, 'neutral'],
  ])(
    'classifies total duration %s independently of output size',
    (seconds, expected) => {
      expect(getResponseTimeColor(seconds, 1)).toBe(expected)
      expect(getResponseTimeColor(seconds, 100_000)).toBe(expected)
    }
  )

  it.each([
    [30, 'success'],
    [15, 'warning'],
    [5, 'orange'],
    [4, 'danger'],
    [Number.NaN, 'neutral'],
  ])('classifies average TPS %s as %s', (tps, expected) => {
    expect(getThroughputColor(tps)).toBe(expected)
  })
})
