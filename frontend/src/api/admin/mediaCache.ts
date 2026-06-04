import { apiClient } from '../client'

export type MediaCacheType = 'all' | 'image' | 'video'

export interface MediaCacheItem {
  id: string
  type: 'image' | 'video' | string
  file_name: string
  path: string
  content_type: string
  size: number
  modified_at: number
  url: string
}

export interface MediaCacheListParams {
  page?: number
  page_size?: number
  type?: MediaCacheType | ''
  search?: string
  before_unix?: number
  older_than?: string
}

export interface MediaCacheListResponse {
  items: MediaCacheItem[]
  total: number
  page: number
  page_size: number
}

export interface MediaCacheCleanupInput {
  type?: MediaCacheType | ''
  before_unix?: number
  older_than?: string
  limit?: number
}

export interface MediaCacheCleanupResult {
  deleted: number
  skipped: number
  errors?: string[]
}

export async function list(params: MediaCacheListParams = {}, options?: { signal?: AbortSignal }): Promise<MediaCacheListResponse> {
  const { data } = await apiClient.get<MediaCacheListResponse>('/admin/media-cache', {
    params,
    signal: options?.signal,
  })
  return data
}

export async function deleteItem(type: string, id: string): Promise<{ deleted: number }> {
  const { data } = await apiClient.delete<{ deleted: number }>(`/admin/media-cache/${encodeURIComponent(type)}/${encodeURIComponent(id)}`)
  return data
}

export async function cleanup(input: MediaCacheCleanupInput): Promise<MediaCacheCleanupResult> {
  const { data } = await apiClient.post<MediaCacheCleanupResult>('/admin/media-cache/cleanup', input)
  return data
}

export async function cleanupOrphans(): Promise<MediaCacheCleanupResult> {
  const { data } = await apiClient.post<MediaCacheCleanupResult>('/admin/media-cache/orphans/cleanup', {})
  return data
}

export const mediaCacheAPI = {
  list,
  deleteItem,
  cleanup,
  cleanupOrphans,
}

export default mediaCacheAPI
