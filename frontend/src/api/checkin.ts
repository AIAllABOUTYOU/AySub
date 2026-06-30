/**
 * User check-in API endpoints.
 */

import { apiClient } from './client'

export interface CheckinRecord {
  checkin_date: string
  quota_awarded: number
  reward_amount?: number
}

export interface CheckinStats {
  total_quota: number
  total_checkins: number
  checkin_count: number
  checked_in_today: boolean
  records: CheckinRecord[]
}

export interface CheckinStatus {
  enabled?: boolean
  checked_in_today: boolean
  reward_amount: number
  checkin_date?: string
  last_checkin_at?: string | null
  next_checkin_at?: string | null
  streak_days?: number
  balance?: number
  stats?: CheckinStats
}

export interface CheckinResult extends CheckinStatus {
  message?: string
  awarded_amount?: number
  new_balance?: number
}

export async function getCheckinStatus(month?: string): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>('/user/checkin', {
    params: month ? { month } : undefined,
  })
  return data
}

export async function submitCheckin(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>('/user/checkin')
  return data
}

export const checkinAPI = {
  getStatus: getCheckinStatus,
  checkin: submitCheckin,
}

export default checkinAPI
