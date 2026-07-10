import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ModelMarketplaceView from '../ModelMarketplaceView.vue'

const { getPublicMarketplace, getUserGroupRates, checkAuth } = vi.hoisted(() => ({
  getPublicMarketplace: vi.fn(),
  getUserGroupRates: vi.fn(),
  checkAuth: vi.fn(),
}))

vi.mock('@/api/channels', () => ({
  default: {
    getAvailable: vi.fn(),
    getPublicMarketplace,
  },
}))

vi.mock('@/api/groups', () => ({
  default: {
    getUserGroupRates,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    showError: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAuthenticated: false,
    checkAuth,
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>, fallback?: string) => {
        if (key === 'modelMarketplace.pricing.tokenSummary') {
          return `输入 ${params?.input} / 输出 ${params?.output}`
        }
        return fallback ?? key
      },
    }),
  }
})

const simpleStub = { template: '<span><slot /></span>' }

describe('ModelMarketplaceView pricing controls', () => {
  beforeEach(() => {
    getPublicMarketplace.mockReset()
    getUserGroupRates.mockReset()
    checkAuth.mockReset()
    getPublicMarketplace.mockResolvedValue([
      {
        name: 'Anthropic channel',
        description: '',
        platforms: [
          {
            platform: 'anthropic',
            groups: [
              {
                id: 1,
                name: 'Standard',
                platform: 'anthropic',
                subscription_type: 'standard',
                rate_multiplier: 2,
                is_exclusive: false,
              },
            ],
            supported_models: [
              {
                name: 'claude-test',
                platform: 'anthropic',
                pricing: {
                  billing_mode: 'token',
                  input_price: 0.00001,
                  output_price: 0.00005,
                  cache_write_price: null,
                  cache_read_price: null,
                  image_output_price: null,
                  per_request_price: null,
                  intervals: [],
                },
              },
            ],
          },
        ],
      },
    ])
  })

  it('updates displayed token prices for M/K and multiplier toggles', async () => {
    const wrapper = mount(ModelMarketplaceView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          PublicHeader: simpleStub,
          Icon: simpleStub,
          PlatformIcon: simpleStub,
          GroupBadge: simpleStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('输入 $10 / 输出 $50')

    await wrapper.get('[data-testid="marketplace-token-unit-toggle"]').trigger('click')
    expect(wrapper.text()).toContain('输入 $0.01 / 输出 $0.05')

    await wrapper.get('[data-testid="marketplace-multiplier-toggle"]').trigger('click')
    expect(wrapper.text()).toContain('输入 $0.02 / 输出 $0.1')
    expect(wrapper.get('[data-testid="marketplace-multiplier-toggle"]').attributes('aria-pressed')).toBe('true')
  })
})
