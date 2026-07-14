import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn() })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    grok: {
      generateAuthUrl: vi.fn(),
      exchangeCode: vi.fn(),
      refreshGrokToken: vi.fn()
    }
  }
}))

import { useGrokOAuth } from '@/composables/useGrokOAuth'

describe('useGrokOAuth.buildCredentials', () => {
  it('persists the Grok CLI subscription proxy for OAuth inference', () => {
    const credentials = useGrokOAuth().buildCredentials({
      access_token: 'access-token',
      token_type: 'Bearer',
      expires_at: 1_900_000_000,
      client_id: 'client-id',
      scope: 'openid grok-cli:access'
    })

    expect(credentials.base_url).toBe('https://cli-chat-proxy.grok.com/v1')
  })
})
