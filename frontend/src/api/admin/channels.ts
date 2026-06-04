/**
 * Admin Channels API endpoints
 * Handles channel management for administrators
 */

import { apiClient } from '../client'
import type { BillingMode, ChannelStatus, BillingModelSource } from '@/constants/channel'

export type { BillingMode } from '@/constants/channel'

export interface PricingInterval {
  id?: number
  min_tokens: number
  max_tokens: number | null
  tier_label: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
  sort_order: number
}

export interface ChannelModelPricing {
  id?: number
  platform: string
  models: string[]
  billing_mode: BillingMode
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: PricingInterval[]
}

export interface AccountStatsPricingRule {
  id?: number
  name: string
  group_ids: number[]
  account_ids: number[]
  pricing: ChannelModelPricing[]
}

export interface Channel {
  id: number
  name: string
  description: string
  status: ChannelStatus
  billing_model_source: BillingModelSource
  restrict_models: boolean
  features_config?: Record<string, unknown>
  group_ids: number[]
  model_pricing: ChannelModelPricing[]
  model_mapping: Record<string, Record<string, string>> // platform → {src→dst}
  apply_pricing_to_account_stats: boolean
  account_stats_pricing_rules: AccountStatsPricingRule[]
  created_at: string
  updated_at: string
}

export interface ChannelStrategyGroup {
  id: number
  name: string
  platform: string
  status: string
  rate_multiplier: number
  account_count: number
  active_account_count: number
  priority_min?: number
  priority_max?: number
}

export interface ChannelStrategyRow {
  channel_id: number
  channel_name: string
  description: string
  status: ChannelStatus
  billing_model_source: BillingModelSource
  restrict_models: boolean
  group_count: number
  groups: ChannelStrategyGroup[]
  platforms: string[]
  model_mapping_count: number
  model_pricing_count: number
  pricing_model_count: number
  billing_modes: string[]
  model_samples: string[]
  apply_pricing_to_account_stats: boolean
  account_stats_pricing_rules_count: number
  request_count: number
  success_count: number
  error_count: number
  error_rate: number
  avg_duration_ms: number
  actual_cost: number
  account_cost: number
  last_error_at?: string
  last_error?: string
}

export interface ChannelStrategyView {
  items: ChannelStrategyRow[]
  start_time: string
  end_time: string
}

export interface CreateChannelRequest {
  name: string
  description?: string
  group_ids?: number[]
  model_pricing?: ChannelModelPricing[]
  model_mapping?: Record<string, Record<string, string>>
  billing_model_source?: string
  restrict_models?: boolean
  features_config?: Record<string, unknown>
  apply_pricing_to_account_stats?: boolean
  account_stats_pricing_rules?: AccountStatsPricingRule[]
}

export interface UpdateChannelRequest {
  name?: string
  description?: string
  status?: string
  group_ids?: number[]
  model_pricing?: ChannelModelPricing[]
  model_mapping?: Record<string, Record<string, string>>
  billing_model_source?: string
  restrict_models?: boolean
  features_config?: Record<string, unknown>
  apply_pricing_to_account_stats?: boolean
  account_stats_pricing_rules?: AccountStatsPricingRule[]
}

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

/**
 * List channels with pagination
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<Channel>> {
  const { data } = await apiClient.get<PaginatedResponse<Channel>>('/admin/channels', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get channel by ID
 */
export async function getById(id: number): Promise<Channel> {
  const { data } = await apiClient.get<Channel>(`/admin/channels/${id}`)
  return data
}

/**
 * Create a new channel
 */
export async function create(req: CreateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.post<Channel>('/admin/channels', req)
  return data
}

/**
 * Update a channel
 */
export async function update(id: number, req: UpdateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.put<Channel>(`/admin/channels/${id}`, req)
  return data
}

/**
 * Delete a channel
 */
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/channels/${id}`)
}

export async function getStrategy(params?: {
  start_date?: string
  end_date?: string
}): Promise<ChannelStrategyView> {
  const { data } = await apiClient.get<ChannelStrategyView>('/admin/channels/strategy', {
    params
  })
  return data
}

export async function batchUpdateStatus(channelIds: number[], status: ChannelStatus): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>('/admin/channels/batch-status', {
    channel_ids: channelIds,
    status
  })
  return data
}

export async function batchReplacePricing(
  channelIds: number[],
  modelPricing: ChannelModelPricing[]
): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>('/admin/channels/batch-pricing', {
    channel_ids: channelIds,
    model_pricing: modelPricing
  })
  return data
}

export async function copyStrategy(req: {
  source_channel_id: number
  target_channel_ids: number[]
  copy_model_pricing: boolean
  copy_model_mapping: boolean
  copy_flags: boolean
  copy_account_stats_pricing: boolean
}): Promise<{ updated: number }> {
  const { data } = await apiClient.post<{ updated: number }>('/admin/channels/copy-strategy', req)
  return data
}

export interface ModelDefaultPricing {
  found: boolean
  input_price?: number    // per-token price
  output_price?: number
  cache_write_price?: number
  cache_read_price?: number
  image_output_price?: number
}

export async function getModelDefaultPricing(model: string): Promise<ModelDefaultPricing> {
  const { data } = await apiClient.get<ModelDefaultPricing>('/admin/channels/model-pricing', {
    params: { model }
  })
  return data
}

export interface SyncPricingModelsResult {
  models: string[]
}

/**
 * Fetch the latest model names from the LiteLLM pricing catalog for the given platform
 */
export async function syncPricingModels(platform: string): Promise<SyncPricingModelsResult> {
  const { data } = await apiClient.get<SyncPricingModelsResult>('/admin/channels/pricing/sync-models', {
    params: { platform }
  })
  return data
}

const channelsAPI = {
  list,
  getById,
  create,
  update,
  remove,
  getStrategy,
  batchUpdateStatus,
  batchReplacePricing,
  copyStrategy,
  getModelDefaultPricing,
  syncPricingModels
}
export default channelsAPI
