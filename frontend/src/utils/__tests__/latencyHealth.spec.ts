import { describe, expect, it } from 'vitest'

import { durationSeverity, firstTokenSeverity, formatLatencyDuration } from '../latencyHealth'

describe('latencyHealth', () => {
  it('classifies first-token latency at 10s/30s/60s boundaries', () => {
    expect(firstTokenSeverity(0)).toBe('good')
    expect(firstTokenSeverity(9_999)).toBe('good')
    expect(firstTokenSeverity(10_000)).toBe('warn')
    expect(firstTokenSeverity(29_999)).toBe('warn')
    expect(firstTokenSeverity(30_000)).toBe('slow')
    expect(firstTokenSeverity(59_999)).toBe('slow')
    expect(firstTokenSeverity(60_000)).toBe('critical')
  })

  it('classifies total duration at 1m/3m/5m boundaries', () => {
    expect(durationSeverity(0)).toBe('good')
    expect(durationSeverity(59_999)).toBe('good')
    expect(durationSeverity(60_000)).toBe('warn')
    expect(durationSeverity(179_999)).toBe('warn')
    expect(durationSeverity(180_000)).toBe('slow')
    expect(durationSeverity(299_999)).toBe('slow')
    expect(durationSeverity(300_000)).toBe('critical')
  })

  it('formats long durations without forcing users to convert seconds', () => {
    expect(formatLatencyDuration(null)).toBe('-')
    expect(formatLatencyDuration(999)).toBe('999ms')
    expect(formatLatencyDuration(1_250)).toBe('1.25s')
    expect(formatLatencyDuration(61_000)).toBe('1m 1s')
    expect(formatLatencyDuration(3_660_000)).toBe('1h 1m')
  })
})
