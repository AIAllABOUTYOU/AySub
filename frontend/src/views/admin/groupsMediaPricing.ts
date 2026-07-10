import type { GroupPlatform } from '@/types'

export type VideoPriceTier =
  | 'video_price_480p'
  | 'video_price_720p'
  | 'video_price_1080p'

const xaiVideoPrices: Record<VideoPriceTier, number> = {
  video_price_480p: 0.05,
  video_price_720p: 0.07,
  video_price_1080p: 0.25,
}

export const supportsVideoPricingPlatform = (platform: GroupPlatform): boolean =>
  platform === 'xai'

export const getDefaultVideoPrice = (
  platform: GroupPlatform,
  tier: VideoPriceTier,
): number | null => (supportsVideoPricingPlatform(platform) ? xaiVideoPrices[tier] : null)

export const getVideoPricePlaceholder = (
  platform: GroupPlatform,
  tier: VideoPriceTier,
): string => getDefaultVideoPrice(platform, tier)?.toString() ?? ''
