import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const { listAccounts, listWithEtag, getBatchTodayStats, getAllProxies, getAllGroups } = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn(), showInfo: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token' })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-test="'scheduler-score-' + row.id">
        <slot name="cell-scheduler_score" :row="row" />
      </div>
    </div>
  `
}

const mountView = () => mount(AccountsView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
      DataTable: DataTableStub,
      HelpTooltip: true,
      Pagination: true,
      ConfirmDialog: true,
      AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
      AccountTableFilters: true,
      AccountBulkActionsBar: true,
      AccountActionMenu: true,
      ImportDataModal: true,
      XaiCookieTokenImportModal: true,
      CodexSessionImportModal: true,
      ReAuthAccountModal: true,
      AccountTestModal: true,
      AccountStatsModal: true,
      ScheduledTestsPanel: true,
      SyncFromCrsModal: true,
      TempUnschedStatusModal: true,
      ErrorPassthroughRulesModal: true,
      TLSFingerprintProfilesModal: true,
      CreateAccountModal: true,
      EditAccountModal: true,
      BulkEditAccountModal: true,
      PlatformTypeBadge: true,
      AccountCapacityCell: true,
      AccountStatusIndicator: true,
      AccountTodayStatsCell: true,
      AccountGroupsCell: true,
      AccountUsageCell: true,
      Icon: true
    }
  }
})

const baseAccount = {
  platform: 'openai',
  type: 'apikey',
  status: 'active',
  schedulable: true,
  concurrency: 1,
  priority: 0,
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z'
}

describe('admin AccountsView scheduler score', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset().mockResolvedValue({
      items: [
        { ...baseAccount, id: 1, name: 'ungrouped', scheduler_score: { base_score: 1.25, sticky_score: 1.25, sticky_score_infinity: true, sticky_weighted_enabled: false } },
        { ...baseAccount, id: 2, name: 'grouped', scheduler_scores: [{ group_id: 5, group_name: 'group-five', base_score: 2, sticky_score: 3, sticky_weighted_enabled: true }] }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockReset().mockResolvedValue({ stats: {} })
    getAllProxies.mockReset().mockResolvedValue([])
    getAllGroups.mockReset().mockResolvedValue([])
  })

  it('keeps expensive score calculation disabled by default', async () => {
    mountView()
    await flushPromises()

    expect(listAccounts.mock.calls[0]?.[2]).toEqual(expect.objectContaining({ include_scheduler_score: '0' }))
  })

  it('requests scores only when a migrated layout explicitly shows the column', async () => {
    localStorage.setItem('account-hidden-columns', JSON.stringify(['today_stats']))
    localStorage.setItem('account-hidden-columns-version', 'scheduler-score-hidden-by-default')

    mountView()
    await flushPromises()

    expect(listAccounts.mock.calls[0]?.[2]).toEqual(expect.objectContaining({ include_scheduler_score: '1' }))
  })

  it('renders ungrouped fallback and per-group scores', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-test="scheduler-score-1"]').text()).toContain('admin.accounts.schedulerScore.ungrouped')
    expect(wrapper.get('[data-test="scheduler-score-1"]').text()).toContain('+∞')
    expect(wrapper.get('[data-test="scheduler-score-2"]').text()).toContain('group-five')
  })
})
