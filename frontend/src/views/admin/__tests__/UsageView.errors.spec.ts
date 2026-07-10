import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const mocks = vi.hoisted(() => ({
  listUsage: vi.fn(),
  getStats: vi.fn(),
  getSnapshotV2: vi.fn(),
  getModelStats: vi.fn(),
  listRequestErrors: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: { list: mocks.listUsage, getStats: mocks.getStats },
    dashboard: { getSnapshotV2: mocks.getSnapshotV2, getModelStats: mocks.getModelStats },
    users: { getById: vi.fn() },
  },
}))

vi.mock('@/api/admin/usage', () => ({ adminUsageAPI: { list: vi.fn() } }))
vi.mock('@/api/admin/ops', () => ({ listRequestErrors: mocks.listRequestErrors }))
vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})
vi.mock('vue-router', () => ({ useRoute: () => ({ query: {} }) }))
vi.mock('@/utils/format', () => ({ formatReasoningEffort: (value: string | null | undefined) => value ?? '-' }))

describe('admin UsageView errors tab', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mocks.listUsage.mockResolvedValue({ items: [], total: 0 })
    mocks.getStats.mockResolvedValue({})
    mocks.getSnapshotV2.mockResolvedValue({ trend: [], groups: [] })
    mocks.getModelStats.mockResolvedValue({ models: [] })
    mocks.listRequestErrors.mockResolvedValue({ items: [], total: 0 })
  })

  afterEach(() => vi.useRealTimers())

  it('forwards shared filters to the request-error API', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          UsageStatsCards: true,
          UsageFilters: { template: '<div><slot name="after-reset" /></div>' },
          UsageTable: true,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: true,
          GroupDistributionChart: true,
          EndpointDistributionChart: true,
          UserTokenRanking: true,
          OpsErrorLogTable: true,
          OpsErrorDetailModal: true,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()
    const vm = wrapper.vm as unknown as { filters: Record<string, unknown> }
    vm.filters.model = 'gpt-5.6-sol'
    vm.filters.account_id = 7
    vm.filters.group_id = 3

    await wrapper.findAll('[data-testid="usage-detail-tab"]')[1].trigger('click')
    await flushPromises()

    expect(mocks.listRequestErrors).toHaveBeenCalledWith(expect.objectContaining({
      view: 'all',
      model: 'gpt-5.6-sol',
      account_id: 7,
      group_id: 3,
      sort_by: 'created_at',
      sort_order: 'desc',
    }))
  })
})
