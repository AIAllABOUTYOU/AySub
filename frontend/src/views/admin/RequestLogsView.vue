<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
        <div class="space-y-4">
          <OperationalReportsPanel
            :reports="reports"
            :loading="reportsLoading"
            :error="reportsError"
            @refresh="loadReports"
          />

          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div class="card p-4">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.stats.total') }}</div>
              <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ pagination.total.toLocaleString() }}</div>
            </div>
            <div class="card p-4">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.stats.success') }}</div>
              <div class="mt-1 text-2xl font-bold text-green-600 dark:text-green-400">{{ pageSuccessCount.toLocaleString() }}</div>
            </div>
            <div class="card p-4">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.stats.errors') }}</div>
              <div class="mt-1 text-2xl font-bold text-red-600 dark:text-red-400">{{ pageErrorCount.toLocaleString() }}</div>
            </div>
            <div class="card p-4">
              <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.stats.errorRate') }}</div>
              <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ pageErrorRate }}</div>
            </div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card space-y-4 p-4">
          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <div>
              <label class="input-label">{{ t('requestLogs.filters.kind') }}</label>
              <Select v-model="filters.kind" :options="kindOptions" @change="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.usage.user') }}</label>
              <input v-model.number="filters.user_id" class="input" type="number" min="1" placeholder="User ID" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
              <input v-model.number="filters.api_key_id" class="input" type="number" min="1" placeholder="Key ID" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.usage.account') }}</label>
              <Select v-model="filters.account_id" :options="accountOptions" searchable @change="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('requestLogs.filters.channel') }}</label>
              <Select v-model="filters.channel_id" :options="channelOptions" searchable @change="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.usage.group') }}</label>
              <Select v-model="filters.group_id" :options="groupOptions" searchable @change="applyFilters" />
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
            <div>
              <label class="input-label">{{ t('requestLogs.filters.platform') }}</label>
              <Select v-model="filters.platform" :options="platformOptions" @change="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('usage.model') }}</label>
              <input v-model.trim="filters.model" class="input" :placeholder="t('requestLogs.filters.modelPlaceholder')" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('usage.endpoint') }}</label>
              <input v-model.trim="filters.endpoint" class="input" placeholder="/v1/responses" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('requestLogs.filters.statusCode') }}</label>
              <input v-model.number="filters.status_code" class="input" type="number" min="0" placeholder="500" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('requestLogs.filters.errorType') }}</label>
              <input v-model.trim="filters.error_type" class="input" placeholder="rate_limit" @keyup.enter="applyFilters" />
            </div>
            <div>
              <label class="input-label">{{ t('requestLogs.filters.errorCode') }}</label>
              <input v-model.trim="filters.error_code" class="input" placeholder="insufficient_quota" @keyup.enter="applyFilters" />
            </div>
          </div>

          <div class="flex flex-wrap items-end gap-3">
            <div class="min-w-[260px] flex-1">
              <label class="input-label">{{ t('common.search') }}</label>
              <input v-model.trim="filters.q" class="input" :placeholder="t('requestLogs.filters.searchPlaceholder')" @keyup.enter="applyFilters" />
            </div>
            <div class="min-w-[280px]">
              <label class="input-label">{{ t('usage.timeRange') }}</label>
              <DateRangePicker
                v-model:start-date="startDate"
                v-model:end-date="endDate"
                @change="onDateRangeChange"
              />
            </div>
            <div class="ml-auto flex items-center gap-2">
              <button class="btn btn-secondary" :disabled="loading" @click="loadLogs">
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                {{ t('common.refresh') }}
              </button>
              <button class="btn btn-secondary" @click="resetFilters">{{ t('common.reset') }}</button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <RequestLogTable
          admin
          :rows="rows"
          :loading="loading"
          @sort="handleSort"
          @openErrorDetail="openErrorDetail"
        />
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <OpsErrorDetailModal
      :show="errorDetailOpen"
      :error-id="selectedErrorId"
      error-type="request"
      @update:show="errorDetailOpen = $event"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import RequestLogTable from '@/components/requestLogs/RequestLogTable.vue'
import OperationalReportsPanel from '@/components/requestLogs/OperationalReportsPanel.vue'
import OpsErrorDetailModal from '@/views/admin/ops/components/OpsErrorDetailModal.vue'
import { adminAPI } from '@/api/admin'
import { opsAPI, type OpsRequestDetail, type OpsRequestDetailsKind, type OpsRequestDetailsParams, type OpsRequestDetailsSort } from '@/api/admin/ops'
import type { OperationalReportsResponse } from '@/api/admin/dashboard'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Account, AdminGroup } from '@/types'
import type { Channel } from '@/api/admin/channels'

const { t } = useI18n()
const appStore = useAppStore()

const rows = ref<OpsRequestDetail[]>([])
const loading = ref(false)
const reports = ref<OperationalReportsResponse | null>(null)
const reportsLoading = ref(false)
const reportsError = ref('')
const accounts = ref<Account[]>([])
const channels = ref<Channel[]>([])
const groups = ref<AdminGroup[]>([])
let abortController: AbortController | null = null

const errorDetailOpen = ref(false)
const selectedErrorId = ref<number | null>(null)

function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const now = new Date()
const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)
const startDate = ref(formatLocalDate(yesterday))
const endDate = ref(formatLocalDate(now))

const filters = reactive<{
  kind: OpsRequestDetailsKind
  user_id: number | null
  api_key_id: number | null
  account_id: number | null
  channel_id: number | null
  group_id: number | null
  platform: string | null
  model: string
  endpoint: string
  status_code: number | null
  error_type: string
  error_code: string
  q: string
}>({
  kind: 'all',
  user_id: null,
  api_key_id: null,
  account_id: null,
  channel_id: null,
  group_id: null,
  platform: null,
  model: '',
  endpoint: '',
  status_code: null,
  error_type: '',
  error_code: '',
  q: '',
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sort = ref<OpsRequestDetailsSort>('created_at_desc')

const kindOptions = computed(() => [
  { value: 'all', label: t('requestLogs.kind.all') },
  { value: 'success', label: t('requestLogs.kind.success') },
  { value: 'error', label: t('requestLogs.kind.error') },
])

const platformOptions = computed(() => [
  { value: null, label: t('requestLogs.filters.allPlatforms') },
  ...Array.from(new Set([
    ...groups.value.map((group) => group.platform).filter(Boolean),
    ...accounts.value.map((account) => account.platform).filter(Boolean),
  ])).sort().map((platform) => ({ value: platform, label: platform.toUpperCase() })),
])

const accountOptions = computed(() => [
  { value: null, label: t('admin.usage.allAccounts') },
  ...accounts.value.map((account) => ({ value: account.id, label: `${account.name || `#${account.id}`} · ${account.platform}` })),
])

const channelOptions = computed(() => [
  { value: null, label: t('requestLogs.filters.allChannels') },
  ...channels.value.map((channel) => ({ value: channel.id, label: channel.name })),
])

const groupOptions = computed(() => [
  { value: null, label: t('admin.usage.allGroups') },
  ...groups.value.map((group) => ({ value: group.id, label: `${group.name} · ${group.platform}` })),
])

const pageSuccessCount = computed(() => rows.value.filter((row) => row.kind === 'success').length)
const pageErrorCount = computed(() => rows.value.filter((row) => row.kind === 'error').length)
const pageErrorRate = computed(() => {
  const total = rows.value.length
  if (!total) return '0.00%'
  return `${((pageErrorCount.value / total) * 100).toFixed(2)}%`
})

function toDayStartIso(date: string): string {
  return new Date(`${date}T00:00:00`).toISOString()
}

function toDayEndIso(date: string): string {
  return new Date(`${date}T23:59:59.999`).toISOString()
}

function positiveNumber(value: number | null): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : undefined
}

function buildParams(): OpsRequestDetailsParams {
  const params: OpsRequestDetailsParams = {
    start_time: toDayStartIso(startDate.value),
    end_time: toDayEndIso(endDate.value),
    page: pagination.page,
    page_size: pagination.page_size,
    kind: filters.kind,
    sort: sort.value,
  }
  if (filters.platform) params.platform = filters.platform
  const userId = positiveNumber(filters.user_id)
  const apiKeyId = positiveNumber(filters.api_key_id)
  const accountId = positiveNumber(filters.account_id)
  const channelId = positiveNumber(filters.channel_id)
  const groupId = positiveNumber(filters.group_id)
  if (userId) params.user_id = userId
  if (apiKeyId) params.api_key_id = apiKeyId
  if (accountId) params.account_id = accountId
  if (channelId) params.channel_id = channelId
  if (groupId) params.group_id = groupId
  if (filters.model) params.model = filters.model
  if (filters.endpoint) params.endpoint = filters.endpoint
  if (typeof filters.status_code === 'number' && Number.isFinite(filters.status_code)) params.status_code = filters.status_code
  if (filters.error_type) params.error_type = filters.error_type
  if (filters.error_code) params.error_code = filters.error_code
  if (filters.q) params.q = filters.q
  return params
}

async function loadLookups() {
  try {
    const [accountRes, channelRes, groupRes] = await Promise.all([
      adminAPI.accounts.list(1, 200, { sort_by: 'name', sort_order: 'asc' }),
      adminAPI.channels.list(1, 200, { sort_by: 'name', sort_order: 'asc' }),
      adminAPI.groups.list(1, 200, { sort_by: 'name', sort_order: 'asc' }),
    ])
    accounts.value = accountRes.items || []
    channels.value = channelRes.items || []
    groups.value = groupRes.items || []
  } catch (err) {
    console.error('[AdminRequestLogs] failed to load lookups', err)
  }
}

async function loadLogs() {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await opsAPI.listRequestDetails(buildParams())
    if (controller.signal.aborted) return
    rows.value = res.items || []
    pagination.total = res.total || 0
    pagination.pages = res.pages || 0
  } catch (err) {
    if (controller.signal.aborted) return
    appStore.showError(extractApiErrorMessage(err, t('requestLogs.failedToLoad')))
    rows.value = []
    pagination.total = 0
    pagination.pages = 0
  } finally {
    if (abortController === controller) loading.value = false
  }
}

async function loadReports() {
  reportsLoading.value = true
  reportsError.value = ''
  try {
    reports.value = await adminAPI.dashboard.getOperationalReports({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: 'day',
      limit: 10,
    })
  } catch (err) {
    reports.value = null
    reportsError.value = extractApiErrorMessage(err, t('requestLogs.reports.failedToLoad'))
  } finally {
    reportsLoading.value = false
  }
}

function applyFilters() {
  pagination.page = 1
  loadLogs()
  loadReports()
}

function resetFilters() {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  startDate.value = formatLocalDate(start)
  endDate.value = formatLocalDate(end)
  filters.kind = 'all'
  filters.user_id = null
  filters.api_key_id = null
  filters.account_id = null
  filters.channel_id = null
  filters.group_id = null
  filters.platform = null
  filters.model = ''
  filters.endpoint = ''
  filters.status_code = null
  filters.error_type = ''
  filters.error_code = ''
  filters.q = ''
  pagination.page = 1
  sort.value = 'created_at_desc'
  loadLogs()
  loadReports()
}

function onDateRangeChange(range: { startDate: string; endDate: string }) {
  startDate.value = range.startDate
  endDate.value = range.endDate
  applyFilters()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadLogs()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadLogs()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  if (key === 'created_at') {
    sort.value = order === 'desc' ? 'created_at_desc' : 'created_at_desc'
  }
  pagination.page = 1
  loadLogs()
}

function openErrorDetail(errorId: number) {
  selectedErrorId.value = errorId
  errorDetailOpen.value = true
}

onMounted(() => {
  loadLookups()
  loadReports()
  loadLogs()
})
</script>
