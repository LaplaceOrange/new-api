import { describe, expect, it } from 'vitest'

import { getSensitiveWordMatches, parseLogOther } from '../format'

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
    expect(getSensitiveWordMatches(parseLogOther('{"sensitive_words":"xxx"}'))).toEqual([])
  })
})
