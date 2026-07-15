import { describe, it, expect } from 'vitest'
import {
  applyInterceptWarmup,
  applyPlanType,
  buildPlanTypeOptions,
  planTypeDisplayLabel,
  readPlanType
} from '../credentialsBuilder'

describe('applyInterceptWarmup', () => {
  it('create + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, true, 'create')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('create + enabled=false: should not add the field', () => {
    const creds: Record<string, unknown> = { access_token: 'tok' }
    applyInterceptWarmup(creds, false, 'create')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=true: should set intercept_warmup_requests to true', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, true, 'edit')
    expect(creds.intercept_warmup_requests).toBe(true)
  })

  it('edit + enabled=false + field exists: should delete the field', () => {
    const creds: Record<string, unknown> = { api_key: 'sk', intercept_warmup_requests: true }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('edit + enabled=false + field absent: should not throw', () => {
    const creds: Record<string, unknown> = { api_key: 'sk' }
    applyInterceptWarmup(creds, false, 'edit')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })

  it('should not affect other fields', () => {
    const creds: Record<string, unknown> = {
      api_key: 'sk',
      base_url: 'url',
      intercept_warmup_requests: true
    }
    applyInterceptWarmup(creds, false, 'edit')
    expect(creds.api_key).toBe('sk')
    expect(creds.base_url).toBe('url')
    expect('intercept_warmup_requests' in creds).toBe(false)
  })
})

describe('plan_type helpers', () => {
  it('maps aliases and reads only string values', () => {
    expect(planTypeDisplayLabel('chatgptpro')).toBe('Pro')
    expect(planTypeDisplayLabel('team')).toBe('Team')
    expect(readPlanType({ plan_type: 'plus' })).toBe('plus')
    expect(readPlanType({ plan_type: 42 })).toBe('')
  })

  it('keeps a current alias or custom value in options without duplicates', () => {
    expect(buildPlanTypeOptions('chatgptpro', 'Clear').filter((option) => option.label === 'Pro')).toHaveLength(1)
    expect(buildPlanTypeOptions('team', 'Clear')).toContainEqual({ value: 'team', label: 'Team' })
  })

  it('sets or clears plan_type without changing other credentials', () => {
    expect(applyPlanType({ email: 'a@example.com' }, ' pro ')).toEqual({ email: 'a@example.com', plan_type: 'pro' })
    expect(applyPlanType({ email: 'a@example.com', plan_type: 'pro' }, '')).toEqual({ email: 'a@example.com' })
  })
})
