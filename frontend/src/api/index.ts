/**
 * API Client for AySub Backend
 * Central export point for all API modules
 */

// Re-export the HTTP client
export { apiClient } from './client'

// Auth API
export { authAPI, isTotp2FARequired, type LoginResponse } from './auth'

// User APIs
export { keysAPI } from './keys'
export { usageAPI } from './usage'
export { userAPI } from './user'
export { redeemAPI, type RedeemHistoryItem } from './redeem'
export { checkinAPI } from './checkin'
export { paymentAPI } from './payment'
export { userGroupsAPI } from './groups'
export { userChannelsAPI } from './channels'
export { totpAPI } from './totp'
export { default as announcementsAPI } from './announcements'
export { channelMonitorUserAPI } from './channelMonitor'
export { statusAPI } from './status'
export {
  securityAPI,
  type AdminSecurityAuditFilterParams,
  type AdminSecurityEntityFilterParams,
  type SecurityAuditFilterParams,
  type SecurityAuditIntegrityResult,
  type SecurityAuditLog,
  type SecurityAuditPaginatedResponse,
  type SecurityIncident,
  type SecurityPolicyRule,
  type SecurityPolicyRuleInput,
  type SecuritySubjectLock,
  type SecuritySubjectLockInput,
  type UserSecurityEventFilterParams,
} from './security'

// Admin APIs
export { adminAPI } from './admin'

// Default export
export { default } from './client'
