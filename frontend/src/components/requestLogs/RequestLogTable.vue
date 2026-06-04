<template>
  <DataTable
    :columns="columns"
    :data="rows"
    :loading="loading"
    :server-side-sort="true"
    :row-key="rowKey"
    default-sort-key="created_at"
    default-sort-order="desc"
    @sort="handleSort"
  >
    <template #cell-kind="{ row }">
      <span :class="['inline-flex items-center rounded-full px-2 py-1 text-xs font-bold', kindBadgeClass(row.kind)]">
        {{ row.kind === 'error' ? t('requestLogs.kind.error') : t('requestLogs.kind.success') }}
      </span>
    </template>

    <template #cell-status="{ row }">
      <span :class="['inline-flex items-center rounded-md px-2 py-1 text-xs font-bold ring-1 ring-inset', statusBadgeClass(row.status_code, row.kind)]">
        {{ row.status_code ?? '-' }}
      </span>
    </template>

    <template #cell-model="{ row }">
      <div class="max-w-[260px] space-y-1">
        <div class="truncate font-mono text-xs font-medium text-gray-900 dark:text-white" :title="displayRequestedModel(row)">
          {{ displayRequestedModel(row) }}
        </div>
        <div v-if="displayUpstreamModel(row)" class="truncate font-mono text-[11px] text-gray-500 dark:text-gray-400" :title="displayUpstreamModel(row)">
          {{ t('usage.upstreamModel') }}: {{ displayUpstreamModel(row) }}
        </div>
      </div>
    </template>

    <template #cell-endpoint="{ row }">
      <div class="max-w-[300px] space-y-1">
        <div class="break-all font-mono text-xs text-gray-700 dark:text-gray-200">
          {{ row.inbound_endpoint || row.request_path || '-' }}
        </div>
        <div v-if="row.upstream_endpoint" class="break-all font-mono text-[11px] text-gray-500 dark:text-gray-400">
          {{ t('usage.upstream') }}: {{ row.upstream_endpoint }}
        </div>
      </div>
    </template>

    <template #cell-request_id="{ row }">
      <div v-if="row.request_id" class="flex max-w-[260px] items-center gap-2">
        <span class="truncate font-mono text-xs text-gray-700 dark:text-gray-200" :title="row.request_id">
          {{ row.request_id }}
        </span>
        <button
          type="button"
          class="rounded-md p-1 text-gray-400 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-300"
          :title="t('requestLogs.copyRequestId')"
          @click="copyRequestId(row.request_id)"
        >
          <Icon name="copy" size="xs" />
        </button>
      </div>
      <span v-else class="text-gray-400">-</span>
    </template>

    <template #cell-owner="{ row }">
      <div class="max-w-[240px] space-y-1 text-xs">
        <div class="truncate font-medium text-gray-900 dark:text-white" :title="row.user_email || ''">
          {{ row.user_email || formatId('user', row.user_id) }}
        </div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="row.api_key_name || ''">
          {{ row.api_key_name || formatId('key', row.api_key_id) }}
        </div>
      </div>
    </template>

    <template #cell-api_key="{ row }">
      <div class="max-w-[220px] truncate text-xs text-gray-700 dark:text-gray-200" :title="row.api_key_name || ''">
        {{ row.api_key_name || formatId('key', row.api_key_id) }}
      </div>
    </template>

    <template #cell-routing="{ row }">
      <div class="max-w-[260px] space-y-1 text-xs">
        <div class="truncate text-gray-900 dark:text-white" :title="row.channel_name || ''">
          {{ row.channel_name || formatId('channel', row.channel_id) }}
        </div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="row.account_name || ''">
          {{ row.account_name || formatId('account', row.account_id) }}
        </div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="row.group_name || ''">
          {{ row.group_name || formatId('group', row.group_id) }}
        </div>
      </div>
    </template>

    <template #cell-timings="{ row }">
      <div class="space-y-1 text-xs">
        <div class="text-gray-900 dark:text-white">{{ t('usage.duration') }}: {{ formatDuration(row.duration_ms) }}</div>
        <div class="text-gray-500 dark:text-gray-400">{{ t('usage.firstToken') }}: {{ formatDuration(row.first_token_ms) }}</div>
        <div class="text-gray-500 dark:text-gray-400">{{ requestTypeLabel(row.request_type, row.stream) }}</div>
      </div>
    </template>

    <template #cell-cost="{ row }">
      <div class="space-y-1 text-xs">
        <div class="font-medium text-green-600 dark:text-green-400">{{ formatCost(row.actual_cost) }}</div>
        <div class="text-gray-500 dark:text-gray-400">{{ t('usage.original') }}: {{ formatCost(row.total_cost) }}</div>
        <div v-if="admin" class="text-gray-500 dark:text-gray-400">{{ t('usage.accountCost') }}: {{ formatCost(row.account_cost) }}</div>
      </div>
    </template>

    <template #cell-error="{ row }">
      <div class="max-w-[320px] space-y-1 text-xs">
        <div class="truncate font-mono text-gray-700 dark:text-gray-200" :title="row.error_code || ''">
          {{ row.error_code || '-' }}
        </div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="row.message || ''">
          {{ row.message || '-' }}
        </div>
      </div>
    </template>

    <template #cell-client="{ row }">
      <div class="max-w-[280px] space-y-1 text-xs">
        <div class="font-mono text-gray-900 dark:text-white">{{ row.ip_address || '-' }}</div>
        <div class="truncate text-gray-500 dark:text-gray-400" :title="row.user_agent || ''">
          {{ row.user_agent || '-' }}
        </div>
      </div>
    </template>

    <template #cell-created_at="{ value }">
      <span class="whitespace-nowrap text-xs text-gray-600 dark:text-gray-400">{{ formatDateTime(value) }}</span>
    </template>

    <template #cell-actions="{ row }">
      <button
        v-if="admin && row.kind === 'error' && row.error_id"
        type="button"
        class="inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-900/20"
        @click="$emit('openErrorDetail', row.error_id)"
      >
        <Icon name="eye" size="xs" />
        {{ t('requestLogs.viewError') }}
      </button>
      <span v-else class="text-xs text-gray-400">-</span>
    </template>

    <template #empty>
      <div class="flex flex-col items-center">
        <Icon name="inbox" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
        <p class="text-sm font-medium text-gray-600 dark:text-gray-300">{{ t('requestLogs.empty') }}</p>
      </div>
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import type { Column } from '@/components/common/types'
import type { OpsRequestDetail } from '@/api/admin/ops'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  rows: OpsRequestDetail[]
  loading?: boolean
  admin?: boolean
}>()

const emit = defineEmits<{
  (e: 'sort', key: string, order: 'asc' | 'desc'): void
  (e: 'openErrorDetail', errorId: number): void
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const admin = computed(() => props.admin === true)

const columns = computed<Column[]>(() => {
  const base: Column[] = [
    { key: 'kind', label: t('requestLogs.columns.kind'), sortable: false },
    { key: 'status', label: t('requestLogs.columns.status'), sortable: false },
    { key: 'request_id', label: t('requestLogs.columns.requestId'), sortable: false },
  ]

  if (admin.value) {
    base.push(
      { key: 'owner', label: t('requestLogs.columns.owner'), sortable: false },
      { key: 'routing', label: t('requestLogs.columns.routing'), sortable: false },
    )
  } else {
    base.push({ key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false })
  }

  base.push(
    { key: 'model', label: t('requestLogs.columns.model'), sortable: false },
    { key: 'endpoint', label: t('requestLogs.columns.endpoint'), sortable: false },
    { key: 'timings', label: t('requestLogs.columns.timings'), sortable: false },
    { key: 'cost', label: t('requestLogs.columns.cost'), sortable: false },
    { key: 'error', label: t('requestLogs.columns.error'), sortable: false },
  )

  if (admin.value) {
    base.push({ key: 'client', label: t('requestLogs.columns.client'), sortable: false })
  }

  base.push(
    { key: 'created_at', label: t('usage.time'), sortable: true },
    { key: 'actions', label: t('requestLogs.columns.actions'), sortable: false },
  )

  return base
})

function rowKey(row: OpsRequestDetail): string {
  return `${row.kind}:${row.error_id ?? row.request_id}:${row.created_at}`
}

function handleSort(key: string, order: 'asc' | 'desc') {
  emit('sort', key, order)
}

function displayRequestedModel(row: OpsRequestDetail): string {
  return row.requested_model || row.model || '-'
}

function displayUpstreamModel(row: OpsRequestDetail): string {
  const upstream = row.upstream_model || ''
  return upstream && upstream !== displayRequestedModel(row) ? upstream : ''
}

function formatDuration(ms: number | null | undefined): string {
  if (typeof ms !== 'number' || !Number.isFinite(ms)) return '-'
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

function formatCost(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '$0.000000'
  return `$${value.toFixed(6)}`
}

function formatId(prefix: string, value: number | null | undefined): string {
  return typeof value === 'number' && value > 0 ? `${prefix}#${value}` : '-'
}

function requestTypeLabel(type: string | null | undefined, stream: boolean | undefined): string {
  const normalized = String(type || '').toLowerCase()
  if (normalized === 'ws_v2') return t('usage.ws')
  if (normalized === 'stream') return t('usage.stream')
  if (normalized === 'sync') return t('usage.sync')
  if (stream) return t('usage.stream')
  return t('usage.unknown')
}

function kindBadgeClass(kind: string): string {
  if (kind === 'error') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
}

function statusBadgeClass(statusCode: number | null | undefined, kind: string): string {
  if (kind === 'success') return 'bg-green-50 text-green-700 ring-green-200 dark:bg-green-900/20 dark:text-green-300 dark:ring-green-800'
  if (typeof statusCode !== 'number') return 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-900/20 dark:text-red-300 dark:ring-red-800'
  if (statusCode >= 500) return 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-900/20 dark:text-red-300 dark:ring-red-800'
  if (statusCode >= 400) return 'bg-amber-50 text-amber-700 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-300 dark:ring-amber-800'
  return 'bg-gray-50 text-gray-700 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600'
}

async function copyRequestId(requestId: string) {
  await copyToClipboard(requestId, t('requestLogs.requestIdCopied'))
}
</script>
