<template>
  <section class="card space-y-4 p-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('requestLogs.reports.title') }}
        </h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('requestLogs.reports.description') }}
        </p>
      </div>
      <button type="button" class="btn btn-secondary" :disabled="loading" @click="$emit('refresh')">
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        {{ t('common.refresh') }}
      </button>
    </div>

    <div v-if="loading" class="flex min-h-[220px] items-center justify-center">
      <LoadingSpinner size="lg" />
    </div>

    <div v-else-if="error" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300">
      {{ error }}
    </div>

    <div v-else-if="!reports" class="rounded-lg border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
      {{ t('requestLogs.reports.empty') }}
    </div>

    <template v-else>
      <div class="grid gap-3 md:grid-cols-3">
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.reports.totalRequests') }}</div>
          <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatNumber(reports.total_requests) }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.reports.totalTokens') }}</div>
          <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatNumber(reports.total_tokens) }}</div>
        </div>
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('requestLogs.reports.totalCost') }}</div>
          <div class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ formatCurrency(reports.total_actual_cost) }}</div>
        </div>
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('requestLogs.reports.requestTrend') }}</h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ reports.granularity }}</span>
          </div>
          <div class="flex h-36 items-end gap-1">
            <div
              v-for="point in requestTrend"
              :key="point.date"
              class="flex min-w-0 flex-1 flex-col items-center gap-1"
              :title="`${point.date} · ${formatNumber(point.total_requests)}`"
            >
              <div class="flex h-28 w-full items-end gap-0.5">
                <div class="flex-1 rounded-t bg-green-500" :style="{ height: barHeight(point.success_count, maxRequestCount) }" />
                <div class="flex-1 rounded-t bg-red-500" :style="{ height: barHeight(point.error_count, maxRequestCount) }" />
              </div>
              <span class="w-full truncate text-center text-[10px] text-gray-400">{{ shortBucket(point.date) }}</span>
            </div>
          </div>
        </div>

        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="mb-3 flex items-center justify-between">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('requestLogs.reports.errorTrend') }}</h3>
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('requestLogs.reports.businessLimited') }}</span>
          </div>
          <div class="flex h-36 items-end gap-1">
            <div
              v-for="point in errorTrend"
              :key="point.date"
              class="flex min-w-0 flex-1 flex-col items-center gap-1"
              :title="`${point.date} · ${formatNumber(point.error_count_total)}`"
            >
              <div class="flex h-28 w-full items-end gap-0.5">
                <div class="flex-1 rounded-t bg-orange-500" :style="{ height: barHeight(point.error_count_sla, maxErrorCount) }" />
                <div class="flex-1 rounded-t bg-gray-400" :style="{ height: barHeight(point.business_limited_count, maxErrorCount) }" />
              </div>
              <span class="w-full truncate text-center text-[10px] text-gray-400">{{ shortBucket(point.date) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="grid gap-4 xl:grid-cols-2">
        <RankingTable
          :title="t('requestLogs.reports.modelCostRanking')"
          :headers="[t('usage.model'), t('requestLogs.reports.requests'), t('requestLogs.reports.cost')]"
          :rows="modelRows"
        />
        <RankingTable
          :title="t('requestLogs.reports.channelHealthRanking')"
          :headers="[t('requestLogs.filters.channel'), t('requestLogs.reports.errorRate'), t('requestLogs.reports.requests')]"
          :rows="channelRows"
        />
        <RankingTable
          :title="t('requestLogs.reports.userSpendingRanking')"
          :headers="[t('admin.usage.user'), t('requestLogs.reports.requests'), t('requestLogs.reports.cost')]"
          :rows="userRows"
        />
        <RankingTable
          :title="t('requestLogs.reports.keySpendingRanking')"
          :headers="[t('usage.apiKeyFilter'), t('requestLogs.reports.requests'), t('requestLogs.reports.cost')]"
          :rows="keyRows"
        />
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { formatCurrency, formatNumber } from '@/utils/format'
import type { OperationalReportsResponse } from '@/api/admin/dashboard'

const props = defineProps<{
  reports: OperationalReportsResponse | null
  loading: boolean
  error: string
}>()

defineEmits<{
  refresh: []
}>()

const { t } = useI18n()

interface RankingRow {
  key: string
  values: string[]
  hint?: string
}

const RankingTable = defineComponent({
  name: 'RankingTable',
  props: {
    title: { type: String, required: true },
    headers: { type: Array as PropType<string[]>, required: true },
    rows: { type: Array as PropType<RankingRow[]>, required: true },
  },
  setup(tableProps) {
    return () =>
      h('div', { class: 'rounded-lg border border-gray-200 p-3 dark:border-dark-700' }, [
        h('h3', { class: 'mb-3 text-sm font-semibold text-gray-900 dark:text-white' }, tableProps.title),
        tableProps.rows.length === 0
          ? h('div', { class: 'py-8 text-center text-sm text-gray-500 dark:text-gray-400' }, t('requestLogs.reports.empty'))
          : [
              h('div', { class: 'space-y-2 md:hidden' },
                tableProps.rows.map((row) =>
                  h('div', { key: row.key, class: 'rounded-lg bg-gray-50 p-3 dark:bg-dark-800' }, [
                    h('div', {
                      class: 'break-words text-sm font-medium text-gray-900 dark:text-white',
                      title: row.hint || row.values[0],
                    }, row.values[0] || '-'),
                    h('div', { class: 'mt-2 space-y-1 text-xs' },
                      tableProps.headers.slice(1).map((header, index) =>
                        h('div', { class: 'flex items-start justify-between gap-3' }, [
                          h('span', { class: 'text-gray-500 dark:text-gray-400' }, header),
                          h('span', { class: 'text-right font-medium text-gray-700 dark:text-gray-200' }, row.values[index + 1] || '-'),
                        ]),
                      ),
                    ),
                  ]),
                ),
              ),
              h('div', { class: 'hidden overflow-x-auto md:block' }, [
                h('table', { class: 'w-full min-w-[420px] text-sm' }, [
                  h('thead', [
                    h('tr', { class: 'border-b border-gray-100 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400' },
                      tableProps.headers.map((header) => h('th', { class: 'px-2 py-2 text-left font-medium' }, header)),
                    ),
                  ]),
                  h('tbody',
                    tableProps.rows.map((row) =>
                      h('tr', { key: row.key, class: 'border-b border-gray-50 last:border-0 dark:border-dark-800' },
                        row.values.map((value, index) =>
                          h('td', {
                            class: [
                              'px-2 py-2 align-top',
                              index === 0 ? 'max-w-[220px] truncate font-medium text-gray-900 dark:text-white' : 'whitespace-nowrap text-gray-600 dark:text-gray-300',
                            ],
                            title: index === 0 ? row.hint || value : undefined,
                          }, value),
                        ),
                      ),
                    ),
                  ),
                ]),
              ]),
            ],
      ])
  },
})

const requestTrend = computed(() => props.reports?.request_trend || [])
const errorTrend = computed(() => props.reports?.error_trend || [])

const maxRequestCount = computed(() => Math.max(1, ...requestTrend.value.map((point) => point.total_requests)))
const maxErrorCount = computed(() => Math.max(1, ...errorTrend.value.map((point) => point.error_count_total)))

const modelRows = computed<RankingRow[]>(() =>
  (props.reports?.model_cost_ranking || []).map((row) => ({
    key: `${row.platform || ''}:${row.model}`,
    values: [
      row.platform ? `${row.model} · ${row.platform}` : row.model || '-',
      formatNumber(row.requests),
      formatCurrency(row.actual_cost),
    ],
    hint: row.model,
  })),
)

const channelRows = computed<RankingRow[]>(() =>
  (props.reports?.channel_health_ranking || []).map((row) => ({
    key: String(row.channel_id),
    values: [
      row.channel_name || `#${row.channel_id}`,
      `${(row.error_rate * 100).toFixed(2)}%`,
      `${formatNumber(row.request_count)} · ${Math.round(row.avg_duration_ms)}ms`,
    ],
    hint: row.last_error || row.channel_name,
  })),
)

const userRows = computed<RankingRow[]>(() =>
  (props.reports?.user_spending_ranking || []).map((row) => ({
    key: String(row.user_id),
    values: [row.email || `#${row.user_id}`, formatNumber(row.requests), formatCurrency(row.actual_cost)],
    hint: row.email,
  })),
)

const keyRows = computed<RankingRow[]>(() =>
  (props.reports?.api_key_spending_ranking || []).map((row) => ({
    key: String(row.api_key_id),
    values: [
      row.key_name || `#${row.api_key_id}`,
      formatNumber(row.requests),
      formatCurrency(row.actual_cost),
    ],
    hint: row.user_email,
  })),
)

function barHeight(value: number, max: number): string {
  if (value <= 0) return '2px'
  return `${Math.max(4, Math.round((value / max) * 112))}px`
}

function shortBucket(bucket: string): string {
  if (!bucket) return '-'
  if (bucket.length > 10) return bucket.slice(5)
  return bucket.slice(5)
}
</script>
