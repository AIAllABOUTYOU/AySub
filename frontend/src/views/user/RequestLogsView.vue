<template>
  <AppLayout>
    <TablePageLayout>
      <template #actions>
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
            <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.stats.avgDuration') }}</div>
            <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatDuration(pageAverageDurationMs) }}</div>
          </div>
        </div>
      </template>

      <template #filters>
        <div class="card p-4">
          <div class="flex flex-wrap items-end gap-3">
            <div class="min-w-[170px]">
              <label class="input-label">{{ t('requestLogs.filters.kind') }}</label>
              <Select v-model="filters.kind" :options="kindOptions" @change="applyFilters" />
            </div>
            <div class="min-w-[180px]">
              <label class="input-label">{{ t('usage.apiKeyFilter') }}</label>
              <Select v-model="filters.api_key_id" :options="apiKeyOptions" @change="applyFilters" />
            </div>
            <div class="min-w-[180px]">
              <label class="input-label">{{ t('usage.model') }}</label>
              <input v-model.trim="filters.model" class="input" :placeholder="t('requestLogs.filters.modelPlaceholder')" @keyup.enter="applyFilters" />
            </div>
            <div class="min-w-[190px]">
              <label class="input-label">{{ t('usage.endpoint') }}</label>
              <input v-model.trim="filters.endpoint" class="input" placeholder="/v1/chat/completions" @keyup.enter="applyFilters" />
            </div>
            <div class="min-w-[140px]">
              <label class="input-label">{{ t('requestLogs.filters.statusCode') }}</label>
              <input v-model.number="filters.status_code" class="input" type="number" min="0" placeholder="400" @keyup.enter="applyFilters" />
            </div>
            <div class="min-w-[220px] flex-1">
              <label class="input-label">{{ t('common.search') }}</label>
              <input v-model.trim="filters.q" class="input" :placeholder="t('requestLogs.filters.searchPlaceholder')" @keyup.enter="applyFilters" />
            </div>
            <div>
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
          :rows="rows"
          :loading="loading"
          @sort="handleSort"
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
import keysAPI from '@/api/keys'
import { usageAPI, type UserRequestLogsParams } from '@/api/usage'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import type { ApiKey } from '@/types'
import type { OpsRequestDetail, OpsRequestDetailsKind, OpsRequestDetailsSort } from '@/api/admin/ops'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const rows = ref<OpsRequestDetail[]>([])
const apiKeys = ref<ApiKey[]>([])
const loading = ref(false)
let abortController: AbortController | null = null

const now = new Date()
const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)

function formatLocalDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const startDate = ref(formatLocalDate(yesterday))
const endDate = ref(formatLocalDate(now))

const filters = reactive<{
  kind: OpsRequestDetailsKind
  api_key_id: number | null
  model: string
  endpoint: string
  status_code: number | null
  q: string
}>({
  kind: 'all',
  api_key_id: null,
  model: '',
  endpoint: '',
  status_code: null,
  q: ''
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

const apiKeyOptions = computed(() => [
  { value: null, label: t('usage.allApiKeys') },
  ...apiKeys.value.map((key) => ({ value: key.id, label: key.name }))
])

const pageSuccessCount = computed(() => rows.value.filter((row) => row.kind === 'success').length)
const pageErrorCount = computed(() => rows.value.filter((row) => row.kind === 'error').length)
const pageAverageDurationMs = computed(() => {
  const values = rows.value
    .map((row) => row.duration_ms)
    .filter((value): value is number => typeof value === 'number' && Number.isFinite(value))
  if (!values.length) return 0
  return values.reduce((sum, value) => sum + value, 0) / values.length
})

function toDayStartIso(date: string): string {
  return new Date(`${date}T00:00:00`).toISOString()
}

function toDayEndIso(date: string): string {
  return new Date(`${date}T23:59:59.999`).toISOString()
}

function buildParams(): UserRequestLogsParams {
  const params: UserRequestLogsParams = {
    start_time: toDayStartIso(startDate.value),
    end_time: toDayEndIso(endDate.value),
    page: pagination.page,
    page_size: pagination.page_size,
    kind: filters.kind,
    sort: sort.value,
  }
  if (filters.api_key_id) params.api_key_id = filters.api_key_id
  if (filters.model) params.model = filters.model
  if (filters.endpoint) params.endpoint = filters.endpoint
  if (typeof filters.status_code === 'number' && Number.isFinite(filters.status_code)) params.status_code = filters.status_code
  if (filters.q) params.q = filters.q
  return params
}

async function loadApiKeys() {
  try {
    const res = await keysAPI.list(1, 100, { status: 'active' })
    apiKeys.value = res.items || []
  } catch (err) {
    console.error('[RequestLogs] failed to load API keys', err)
  }
}

async function loadLogs() {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await usageAPI.requestLogs(buildParams(), { signal: controller.signal })
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

function applyFilters() {
  pagination.page = 1
  loadLogs()
}

function resetFilters() {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  startDate.value = formatLocalDate(start)
  endDate.value = formatLocalDate(end)
  filters.kind = 'all'
  filters.api_key_id = null
  filters.model = ''
  filters.endpoint = ''
  filters.status_code = null
  filters.q = ''
  pagination.page = 1
  sort.value = 'created_at_desc'
  loadLogs()
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

function formatDuration(ms: number): string {
  if (!ms) return '-'
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

onMounted(() => {
  loadApiKeys()
  loadLogs()
})
</script>
