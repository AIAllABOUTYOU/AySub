import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PlanEditDialog from '../PlanEditDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'payment.admin.subscriptionCnyPayPreview') return `preview ${params?.amount}`
      if (key === 'payment.admin.subscriptionCnyPayPreviewWithFee') return `fee ${params?.feeRate} ${params?.total}`
      return key
    },
  }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    createPlan: vi.fn(),
    updatePlan: vi.fn(),
  },
}))

function mountDialog(subscriptionRate: number, feeRate = 0) {
  return mount(PlanEditDialog, {
    props: {
      show: true,
      plan: null,
      groups: [],
      paymentConfig: {
        subscription_usd_to_cny_rate: subscriptionRate,
        recharge_fee_rate: feeRate,
      },
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Select: true,
        Icon: true,
        GroupBadge: true,
      },
    },
  })
}

describe('PlanEditDialog subscription CNY payment preview', () => {
  it('shows the converted CNY charge and fee total when enabled', async () => {
    const wrapper = mountDialog(7.15, 2.5)

    await wrapper.find('input[type="number"]').setValue('9.99')

    expect(wrapper.text()).toContain('preview')
    expect(wrapper.text()).toContain('¥71.43')
    expect(wrapper.text()).toContain('fee 2.5')
    expect(wrapper.text()).toContain('¥73.22')
  })

  it('hides the preview when the conversion rate is disabled', async () => {
    const wrapper = mountDialog(0, 2.5)

    await wrapper.find('input[type="number"]').setValue('9.99')

    expect(wrapper.text()).not.toContain('preview')
    expect(wrapper.text()).not.toContain('¥71.43')
  })
})
