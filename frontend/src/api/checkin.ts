/**
 * User check-in API endpoints.
 */

import { apiClient } from './client'

export interface CheckinStatus {
  checked_in_today: boolean
  reward_amount: number
  last_checkin_at?: string | null
  next_checkin_at?: string | null
  streak_days?: number
  balance?: number
}

export interface CheckinResult extends CheckinStatus {
  message?: string
  awarded_amount?: number
  new_balance?: number
}

export async function getCheckinStatus(): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>('/user/checkin')
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
