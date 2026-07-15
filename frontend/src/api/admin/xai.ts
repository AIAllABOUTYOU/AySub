/**
 * Admin xAI/Grok OAuth endpoints.
 */

import { apiClient } from '../client'

export interface GrokAuthUrlResponse {
  auth_url: string
  session_id: string
  state: string
}

export interface GrokAuthUrlRequest {
  proxy_id?: number
  redirect_uri?: string
}

export interface GrokExchangeCodeRequest {
  session_id: string
  state: string
  code: string
  proxy_id?: number
  redirect_uri?: string
}

export interface GrokTokenInfo {
  access_token?: string
  refresh_token?: string
  token_type?: string
  id_token?: string
  expires_at?: number | string
  expires_in?: number
  scope?: string
  client_id?: string
  email?: string
  sub?: string
  team_id?: string
  subscription_tier?: string
  entitlement_status?: string
  [key: string]: unknown
}

export interface GrokSSOToOAuthRequest {
  sso_tokens: string[]
  name?: string
  notes?: string | null
  proxy_id?: number | null
  group_ids?: number[]
  credentials?: Record<string, unknown>
  extra?: Record<string, unknown>
  concurrency?: number
  load_factor?: number
  priority?: number
  rate_multiplier?: number
  expires_at?: number | null
  auto_pause_on_expired?: boolean
}

export interface GrokSSOToOAuthItemResult {
  index: number
  name?: string
  email?: string
  account?: unknown
  error?: string
}

export interface GrokSSOToOAuthResponse {
  created: GrokSSOToOAuthItemResult[]
  failed: GrokSSOToOAuthItemResult[]
}

const GROK_SSO_IMPORT_CONCURRENCY = 3
const GROK_SSO_IMPORT_TIMEOUT_PER_BATCH_MS = 90_000
const GROK_SSO_IMPORT_TIMEOUT_BUFFER_MS = 90_000

export function getGrokSSOImportTimeout(tokenCount: number): number {
  const batches = Math.ceil(Math.max(1, tokenCount) / GROK_SSO_IMPORT_CONCURRENCY)
  return batches * GROK_SSO_IMPORT_TIMEOUT_PER_BATCH_MS + GROK_SSO_IMPORT_TIMEOUT_BUFFER_MS
}

export async function generateAuthUrl(payload: GrokAuthUrlRequest): Promise<GrokAuthUrlResponse> {
  const { data } = await apiClient.post<GrokAuthUrlResponse>('/admin/xai/oauth/auth-url', payload)
  return data
}

export async function exchangeCode(payload: GrokExchangeCodeRequest): Promise<GrokTokenInfo> {
  const { data } = await apiClient.post<GrokTokenInfo>('/admin/xai/oauth/exchange-code', payload)
  return data
}

export async function refreshGrokToken(
  refreshToken: string,
  proxyId?: number | null
): Promise<GrokTokenInfo> {
  const payload: Record<string, unknown> = { refresh_token: refreshToken }
  if (proxyId) payload.proxy_id = proxyId

  const { data } = await apiClient.post<GrokTokenInfo>('/admin/xai/oauth/refresh-token', payload)
  return data
}

export async function createFromSSO(payload: GrokSSOToOAuthRequest): Promise<GrokSSOToOAuthResponse> {
  const { data } = await apiClient.post<GrokSSOToOAuthResponse>(
    '/admin/xai/sso-to-oauth',
    payload,
    { timeout: getGrokSSOImportTimeout(payload.sso_tokens.length) }
  )
  return data
}

export default { generateAuthUrl, exchangeCode, refreshGrokToken, createFromSSO }
