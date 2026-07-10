export const BILLING_MODE_TOKEN = 'token'
export const BILLING_MODE_PER_REQUEST = 'per_request'
export const BILLING_MODE_IMAGE = 'image'
export const BILLING_MODE_VIDEO = 'video'

export function getBillingModeLabel(mode: string | null | undefined, t: (key: string) => string): string {
  switch (mode) {
    case BILLING_MODE_PER_REQUEST: return t('admin.usage.billingModePerRequest')
    case BILLING_MODE_IMAGE: return t('admin.usage.billingModeImage')
    case BILLING_MODE_VIDEO: return t('admin.usage.billingModeVideo')
    default: return t('admin.usage.billingModeToken')
  }
}

export function getBillingModeBadgeClass(mode: string | null | undefined): string {
  switch (mode) {
    case BILLING_MODE_PER_REQUEST: return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300'
    case BILLING_MODE_IMAGE: return 'bg-pink-100 text-pink-700 dark:bg-pink-900/30 dark:text-pink-300'
    case BILLING_MODE_VIDEO: return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    default: return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  }
}

interface MediaBillingRow {
  billing_mode?: string | null
  image_count?: number | null
  video_count?: number | null
  video_resolution?: string | null
  video_duration_seconds?: number | null
}

export function isVideoUsage(row: MediaBillingRow | null | undefined): boolean {
  return row?.billing_mode === BILLING_MODE_VIDEO ||
    (row?.video_count ?? 0) > 0 ||
    !!row?.video_resolution ||
    row?.video_duration_seconds != null
}

export function isImageUsage(row: MediaBillingRow | null | undefined): boolean {
  return !isVideoUsage(row) &&
    (row?.image_count ?? 0) > 0 &&
    row?.billing_mode !== BILLING_MODE_TOKEN
}

export function getDisplayBillingMode(row: MediaBillingRow | null | undefined): string | null | undefined {
  if (isVideoUsage(row)) return BILLING_MODE_VIDEO
  if ((row?.image_count ?? 0) > 0 && !row?.billing_mode) return BILLING_MODE_IMAGE
  return row?.billing_mode
}

interface VideoPriceRow extends MediaBillingRow {
  total_cost?: number | null
}

export function videoUnitPrice(row: VideoPriceRow | null | undefined): number {
  const seconds = row?.video_duration_seconds ?? 0
  const count = (row?.video_count ?? 0) > 0 ? row!.video_count! : (isVideoUsage(row) ? 1 : 0)
  if (seconds <= 0 || count <= 0) return 0
  const price = (row?.total_cost ?? 0) / seconds / count
  return Number.isFinite(price) ? price : 0
}
