import { apiClient } from './client'

export interface PublicStatusModels {
  visible: boolean
  count: number
  names?: string[]
}

export interface PublicStatusChannelSummary {
  platform: string
  total: number
  active: number
  model_count: number
}

export interface PublicStatusChannels {
  visible: boolean
  total: number
  active: number
  disabled_or_error: number
  summaries?: PublicStatusChannelSummary[]
}

export interface PublicStatusLatencyRange {
  avg: number
  min_bucket_avg: number
  max_bucket_avg: number
}

export interface PublicStatusLast24h {
  requests: number
  error_rate: number
  latency_ms: PublicStatusLatencyRange
}

export interface PublicStatusEvent {
  created_at: string
  severity: 'critical' | 'error' | 'warning' | 'info' | string
  summary: string
  endpoint?: string
  status_code?: number
}

export interface PublicStatusResponse {
  enabled: boolean
  status: 'operational' | 'degraded' | string
  generated_at: string
  models: PublicStatusModels
  channels: PublicStatusChannels
  last_24h: PublicStatusLast24h
  recent_events: PublicStatusEvent[]
}

export async function getPublicStatus(options?: { signal?: AbortSignal }): Promise<PublicStatusResponse> {
  const { data } = await apiClient.get<PublicStatusResponse>('/status/public', {
    signal: options?.signal,
  })
  return data
}

export const statusAPI = { getPublicStatus }

export default statusAPI
