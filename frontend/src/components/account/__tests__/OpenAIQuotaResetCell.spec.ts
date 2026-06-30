import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpenAIQuotaResetCell from '../OpenAIQuotaResetCell.vue'
import type { Account } from '@/types'

const { queryOpenAIQuota, resetOpenAIQuota } = vi.hoisted(() => ({
  queryOpenAIQuota: vi.fn(),
  resetOpenAIQuota: vi.fn()
}))

vi.mock('@/api/admin/accounts', () => ({
  queryOpenAIQuota,
  resetOpenAIQuota
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        return `${key}:${JSON.stringify(params)}`
      }
    })
  }
})

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  props: {
    show: { type: Boolean, required: true },
    title: { type: String, required: true },
    message: { type: String, required: true },
    confirmText: { type: String, default: '' },
    cancelText: { type: String, default: '' },
    danger: { type: Boolean, default: false }
  },
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" class="confirm-dialog">
      <span class="dialog-title">{{ title }}</span>
      <span class="dialog-message">{{ message }}</span>
      <button class="dialog-confirm" @click="$emit('confirm')">confirm</button>
      <button class="dialog-cancel" @click="$emit('cancel')">cancel</button>
    </div>
  `
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'openai-oauth',
    platform: 'openai',
    type: 'oauth',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-06-30T00:00:00Z',
    updated_at: '2026-06-30T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

describe('OpenAIQuotaResetCell', () => {
  beforeEach(() => {
    queryOpenAIQuota.mockReset()
    resetOpenAIQuota.mockReset()
  })

  it('requires confirmation before consuming a reset credit', async () => {
    queryOpenAIQuota.mockResolvedValue({
      rate_limit_reset_credits: {
        available_count: 2
      }
    })
    resetOpenAIQuota.mockResolvedValue({
      windows_reset: 1
    })

    const wrapper = mount(OpenAIQuotaResetCell, {
      props: {
        account: makeAccount()
      },
      global: {
        stubs: {
          ConfirmDialog: ConfirmDialogStub
        }
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    await flushPromises()

    expect(queryOpenAIQuota).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('2')

    await buttons[1].trigger('click')
    await flushPromises()

    expect(resetOpenAIQuota).not.toHaveBeenCalled()
    expect(wrapper.find('.confirm-dialog').exists()).toBe(true)
    expect(wrapper.find('.dialog-title').text()).toBe('admin.accounts.openaiQuotaReset.confirmTitle')
    expect(wrapper.find('.dialog-message').text()).toContain('"count":2')

    await wrapper.find('.dialog-confirm').trigger('click')
    await flushPromises()

    expect(resetOpenAIQuota).toHaveBeenCalledWith(1)
  })
})
