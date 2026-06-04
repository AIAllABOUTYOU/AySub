import { apiClient } from './client'
import type { FetchOptions, PaginatedResponse } from '@/types'

export type SecurityAuditResult = 'success' | 'denied' | 'failure' | (string & {})
export type SecurityAuditRiskLevel = 'low' | 'medium' | 'high' | 'critical' | (string & {})
export type SecurityPolicyAction =
  | 'observe'
  | 'block'
  | 'challenge'
  | 'disable_api_key'
  | 'disable_user'
  | 'temporary_lock'
  | 'notify_admin'
  | 'notify_user'
  | (string & {})

export interface SecurityAuditLog {
  id: number
  event_type: string
  actor_type: string
  actor_id?: number | null
  actor_label?: string
  subject_type: string
  subject_id?: number | null
  subject_label?: string
  resource_type: string
  resource_id: string
  action: string
  result: SecurityAuditResult
  risk_level: SecurityAuditRiskLevel
  request_id: string
  ip: string
  user_agent: string
  endpoint: string
  reason: string
  metadata?: Record<string, unknown>
  diff_summary?: Record<string, unknown>
  prev_hash: string
  entry_hash: string
  created_at: string
}

export interface SecurityAuditFilterParams {
  page?: number
  page_size?: number
  action?: string
  result?: SecurityAuditResult | ''
  risk_level?: SecurityAuditRiskLevel | ''
  request_id?: string
  q?: string
  start_time?: string
  end_time?: string
}

export interface UserSecurityEventFilterParams extends SecurityAuditFilterParams {
  actor_type?: string
  actor_id?: number
}

export interface AdminSecurityAuditFilterParams extends SecurityAuditFilterParams {
  actor_type?: string
  actor_id?: number
  subject_type?: string
  subject_id?: number
}

export type SecurityAuditPaginatedResponse = PaginatedResponse<SecurityAuditLog>

export interface SecurityIncident {
  id: number
  incident_key: string
  title: string
  status: string
  severity: SecurityAuditRiskLevel
  subject_type: string
  subject_id?: number | null
  first_audit_log_id?: number | null
  last_audit_log_id?: number | null
  metadata?: Record<string, unknown>
  detected_at: string
  resolved_at?: string | null
  created_at: string
  updated_at: string
}

export interface SecurityPolicyRule {
  id: number
  name: string
  code: string
  description: string
  enabled: boolean
  severity: SecurityAuditRiskLevel
  action: SecurityPolicyAction
  conditions?: Record<string, unknown>
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface SecurityPolicyRuleInput {
  name: string
  code: string
  description?: string
  enabled?: boolean
  severity?: SecurityAuditRiskLevel
  action?: SecurityPolicyAction
  conditions?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface SecuritySubjectLock {
  id: number
  subject_type: string
  subject_id: number
  reason: string
  status: string
  locked_by_type: string
  locked_by_id?: number | null
  audit_log_id?: number | null
  metadata?: Record<string, unknown>
  locked_at: string
  expires_at?: string | null
  unlocked_at?: string | null
  created_at: string
  updated_at: string
}

export interface SecuritySubjectLockInput {
  subject_type: string
  subject_id: number
  reason?: string
  metadata?: Record<string, unknown>
  expires_at?: string | null
}

export interface AdminSecurityEntityFilterParams {
  page?: number
  page_size?: number
  status?: string
  severity?: string
  action?: string
  enabled?: boolean
  subject_type?: string
  subject_id?: number
  q?: string
}

export interface SecurityAuditExport {
  id: number
  export_key: string
  requested_by_type: string
  requested_by_id?: number | null
  status: 'pending' | 'completed' | 'failed' | (string & {})
  filters?: Record<string, unknown>
  file_sha256: string
  row_count: number
  error_message?: string
  requested_at: string
  completed_at?: string | null
  expires_at?: string | null
  created_at: string
  updated_at: string
  download_available: boolean
}

export type SecurityAuditExportInput = Omit<AdminSecurityAuditFilterParams, 'page' | 'page_size'>
export type SecurityAuditExportPaginatedResponse = PaginatedResponse<SecurityAuditExport>

export type SecurityIncidentPaginatedResponse = PaginatedResponse<SecurityIncident>
export type SecurityPolicyPaginatedResponse = PaginatedResponse<SecurityPolicyRule>
export type SecurityLockPaginatedResponse = PaginatedResponse<SecuritySubjectLock>

export interface SecurityAuditIntegrityResult {
  valid: boolean
  checked: number
  broken_at_id?: number | null
  expected_prev_hash?: string
  actual_prev_hash?: string
  expected_hash?: string
  actual_hash?: string
}

export interface SensitiveOperationVerifyInput {
  action: string
  password?: string
  totp_code?: string
  recovery_code?: string
}

export interface SensitiveOperationVerifyResult {
  token: string
  expires_in: number
  action: string
}

export async function listUserSecurityEvents(
  params: UserSecurityEventFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityAuditPaginatedResponse> {
  const { data } = await apiClient.get<SecurityAuditPaginatedResponse>('/user/security/events', {
    signal: options.signal,
    params,
  })
  return data
}

export async function listAdminSecurityAuditLogs(
  params: AdminSecurityAuditFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityAuditPaginatedResponse> {
  const { data } = await apiClient.get<SecurityAuditPaginatedResponse>('/admin/security/audit-logs', {
    signal: options.signal,
    params,
  })
  return data
}

export async function checkAdminSecurityIntegrity(
  options: FetchOptions = {},
): Promise<SecurityAuditIntegrityResult> {
  const { data } = await apiClient.get<SecurityAuditIntegrityResult>('/admin/security/integrity/check', {
    signal: options.signal,
  })
  return data
}

export async function listAdminSecurityIncidents(
  params: AdminSecurityEntityFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityIncidentPaginatedResponse> {
  const { data } = await apiClient.get<SecurityIncidentPaginatedResponse>('/admin/security/incidents', {
    signal: options.signal,
    params,
  })
  return data
}

export async function listAdminSecurityPolicies(
  params: AdminSecurityEntityFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityPolicyPaginatedResponse> {
  const { data } = await apiClient.get<SecurityPolicyPaginatedResponse>('/admin/security/policies', {
    signal: options.signal,
    params,
  })
  return data
}

export async function createAdminSecurityPolicy(
  input: SecurityPolicyRuleInput,
  options: FetchOptions = {},
): Promise<SecurityPolicyRule> {
  const { data } = await apiClient.post<SecurityPolicyRule>('/admin/security/policies', input, {
    signal: options.signal,
  })
  return data
}

export async function updateAdminSecurityPolicy(
  id: number,
  input: SecurityPolicyRuleInput,
  options: FetchOptions = {},
): Promise<SecurityPolicyRule> {
  const { data } = await apiClient.put<SecurityPolicyRule>(`/admin/security/policies/${id}`, input, {
    signal: options.signal,
  })
  return data
}

export async function deleteAdminSecurityPolicy(id: number, options: FetchOptions = {}): Promise<{ success: boolean }> {
  const { data } = await apiClient.delete<{ success: boolean }>(`/admin/security/policies/${id}`, {
    signal: options.signal,
  })
  return data
}

export async function listAdminSecurityLocks(
  params: AdminSecurityEntityFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityLockPaginatedResponse> {
  const { data } = await apiClient.get<SecurityLockPaginatedResponse>('/admin/security/locks', {
    signal: options.signal,
    params,
  })
  return data
}

export async function createAdminSecurityLock(
  input: SecuritySubjectLockInput,
  options: FetchOptions = {},
): Promise<SecuritySubjectLock> {
  const { data } = await apiClient.post<SecuritySubjectLock>('/admin/security/locks', input, {
    signal: options.signal,
  })
  return data
}

export async function unlockAdminSecurityLock(
  id: number,
  reason = '',
  options: FetchOptions = {},
): Promise<SecuritySubjectLock> {
  const { data } = await apiClient.post<SecuritySubjectLock>(`/admin/security/locks/${id}/unlock`, { reason }, {
    signal: options.signal,
  })
  return data
}

export async function listAdminSecurityExports(
  params: AdminSecurityEntityFilterParams = {},
  options: FetchOptions = {},
): Promise<SecurityAuditExportPaginatedResponse> {
  const { data } = await apiClient.get<SecurityAuditExportPaginatedResponse>('/admin/security/exports', {
    signal: options.signal,
    params,
  })
  return data
}

export async function createAdminSecurityExport(
  input: SecurityAuditExportInput,
  options: FetchOptions = {},
): Promise<SecurityAuditExport> {
  const { data } = await apiClient.post<SecurityAuditExport>('/admin/security/exports', input, {
    signal: options.signal,
  })
  return data
}

export async function downloadAdminSecurityExport(id: number, options: FetchOptions = {}): Promise<Blob> {
  const { data } = await apiClient.get<Blob>(`/admin/security/exports/${id}/download`, {
    signal: options.signal,
    responseType: 'blob',
  })
  return data
}

export async function verifySensitiveOperation(
  input: SensitiveOperationVerifyInput,
  options: FetchOptions = {},
): Promise<SensitiveOperationVerifyResult> {
  const { data } = await apiClient.post<SensitiveOperationVerifyResult>('/user/security/verify-sensitive-operation', input, {
    signal: options.signal,
  })
  return data
}

export async function revokeUserSecurityAPIKey(
  id: number,
  sensitiveToken = '',
  options: FetchOptions = {},
): Promise<{ success: boolean }> {
  const { data } = await apiClient.post<{ success: boolean }>(`/user/security/api-keys/${id}/revoke`, {}, {
    signal: options.signal,
    headers: sensitiveToken ? { 'X-Sensitive-Operation-Token': sensitiveToken } : undefined,
  })
  return data
}

export const securityAPI = {
  listUserEvents: listUserSecurityEvents,
  listAdminAuditLogs: listAdminSecurityAuditLogs,
  checkAdminIntegrity: checkAdminSecurityIntegrity,
  listAdminIncidents: listAdminSecurityIncidents,
  listAdminPolicies: listAdminSecurityPolicies,
  createAdminPolicy: createAdminSecurityPolicy,
  updateAdminPolicy: updateAdminSecurityPolicy,
  deleteAdminPolicy: deleteAdminSecurityPolicy,
  listAdminLocks: listAdminSecurityLocks,
  createAdminLock: createAdminSecurityLock,
  unlockAdminLock: unlockAdminSecurityLock,
  listAdminExports: listAdminSecurityExports,
  createAdminExport: createAdminSecurityExport,
  downloadAdminExport: downloadAdminSecurityExport,
  verifySensitiveOperation,
  revokeUserAPIKey: revokeUserSecurityAPIKey,
}

export default securityAPI
