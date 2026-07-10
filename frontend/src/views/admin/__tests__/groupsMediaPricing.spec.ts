import { describe, expect, it } from 'vitest'

import {
  getDefaultVideoPrice,
  getVideoPricePlaceholder,
  supportsVideoPricingPlatform,
} from '../groupsMediaPricing'

describe('groups media pricing', () => {
  it('exposes xAI per-second defaults', () => {
    expect(supportsVideoPricingPlatform('xai')).toBe(true)
    expect(supportsVideoPricingPlatform('openai')).toBe(false)
    expect(getDefaultVideoPrice('xai', 'video_price_480p')).toBe(0.05)
    expect(getDefaultVideoPrice('xai', 'video_price_720p')).toBe(0.07)
    expect(getVideoPricePlaceholder('xai', 'video_price_1080p')).toBe('0.25')
  })
})
