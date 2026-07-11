import { describe, expect, it } from 'vitest'

import { normalizeCustomHeaderRows } from '../customHeaders'

describe('normalizeCustomHeaderRows', () => {
  it('normalizes names and values', () => {
    expect(normalizeCustomHeaderRows([{ key: ' X-Trace-ID ', value: ' abc ' }])).toEqual({
      headers: { 'x-trace-id': 'abc' },
      error: null
    })
  })

  it('rejects blocked and case-insensitive duplicate names', () => {
    expect(normalizeCustomHeaderRows([{ key: 'Authorization', value: 'secret' }]).error).toBe('blockedName')
    expect(normalizeCustomHeaderRows([
      { key: 'X-Test', value: 'one' },
      { key: 'x-test', value: 'two' }
    ]).error).toBe('duplicateName')
  })

  it('rejects invalid names and response-splitting values', () => {
    expect(normalizeCustomHeaderRows([{ key: 'bad name', value: 'x' }]).error).toBe('invalidName')
    expect(normalizeCustomHeaderRows([{ key: 'x-test', value: 'ok\r\nforged: yes' }]).error).toBe('invalidValue')
  })
})
