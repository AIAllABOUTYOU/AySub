import { describe, expect, it } from 'vitest'

import {
  BILLING_MODE_VIDEO,
  getDisplayBillingMode,
  isImageUsage,
  isVideoUsage,
  videoUnitPrice,
} from '@/utils/billingMode'

describe('video billing mode helpers', () => {
  it('prioritizes video metadata over legacy image_count', () => {
    const row = {
      billing_mode: null,
      image_count: 1,
      video_count: 1,
      video_resolution: '720p',
      video_duration_seconds: 20,
      total_cost: 2.8,
    }

    expect(isVideoUsage(row)).toBe(true)
    expect(isImageUsage(row)).toBe(false)
    expect(getDisplayBillingMode(row)).toBe(BILLING_MODE_VIDEO)
    expect(videoUnitPrice(row)).toBeCloseTo(0.14)
  })
})
