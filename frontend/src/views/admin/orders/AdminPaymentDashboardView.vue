<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header with Day Switcher -->
      <div class="flex items-center justify-end">
        <div class="flex items-center gap-2">
          <div class="flex rounded-lg border border-gray-200 dark:border-dark-600">
            <button
              v-for="d in DAYS_OPTIONS"
              :key="d"
              type="button"
              class="px-3 py-1.5 text-xs font-medium transition-colors first:rounded-l-lg last:rounded-r-lg"
              :class="days === d
                ? 'bg-primary-600 text-white'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'"
              @click="days = d"
            >
              {{ d }}{{ t('payment.admin.daySuffix') }}
            </button>
          </div>
          <button @click="loadDashboard" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <!-- Dashboard Content -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>
      <template v-else-if="stats">
        <OrderStatsCards :stats="stats" />
        <DailyRevenueChart :data="stats.daily_series || []" :loading="loading" />
        <section v-if="stats.ops" class="card p-4">
          <div class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('payment.admin.ops.title') }}
              </h3>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('payment.admin.ops.window', { days: stats.ops.window_days }) }}
              </p>
            </div>
            <span class="inline-flex w-fit items-center gap-1 rounded-full bg-gray-100 px-3 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
              <Icon name="shield" size="xs" />
              {{ t('payment.admin.ops.productionReadiness') }}
            </span>
          </div>

          <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
            <div
              v-for="section in opsSections"
              :key="section.title"
              class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
            >
              <div class="mb-3 flex items-center gap-2">
                <span :class="['rounded-lg p-2', section.iconBg]">
                  <Icon :name="section.icon" size="sm" :class="section.iconClass" :stroke-width="2" />
                </span>
                <div class="min-w-0">
                  <h4 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                    {{ section.title }}
                  </h4>
                  <p class="text-xs text-gray-500 dark:text-gray-400">
                    {{ section.description }}
                  </p>
                </div>
              </div>

              <div class="space-y-2">
                <div
                  v-for="item in section.items"
                  :key="item.label"
                  class="flex items-center justify-between gap-3 rounded-md bg-gray-50 px-3 py-2 dark:bg-dark-700/50"
                >
                  <span class="min-w-0 text-xs text-gray-600 dark:text-gray-300">
                    {{ item.label }}
                  </span>
                  <span :class="['shrink-0 text-sm font-semibold tabular-nums', item.className]">
                    {{ item.value.toLocaleString() }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div class="card p-4">
            <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('payment.admin.paymentDistribution') }}</h3>
            <div v-if="!stats.payment_methods?.length" class="flex h-32 items-center justify-center text-sm text-gray-500 dark:text-gray-400">{{ t('payment.admin.noData') }}</div>
            <div v-else class="space-y-3">
              <div v-for="method in stats.payment_methods" :key="method.type" class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span :class="['inline-block h-3 w-3 rounded-full', methodColor(method.type)]"></span>
                  <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('payment.methods.' + method.type, method.type) }}</span>
                </div>
                <div class="text-right">
                  <span class="text-sm font-medium text-gray-900 dark:text-white">&yen;{{ method.amount.toFixed(2) }}</span>
                  <span class="ml-2 text-xs text-gray-500 dark:text-gray-400">({{ method.count }})</span>
                </div>
              </div>
            </div>
          </div>
          <div class="card p-4">
            <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('payment.admin.topUsers') }}</h3>
            <div v-if="!stats.top_users?.length" class="flex h-32 items-center justify-center text-sm text-gray-500 dark:text-gray-400">{{ t('payment.admin.noData') }}</div>
            <div v-else class="space-y-2">
              <div v-for="(user, idx) in stats.top_users" :key="user.user_id" class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-gray-50 dark:hover:bg-dark-700">
                <div class="flex min-w-0 items-center gap-3">
                  <span :class="['flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold', rankClass(idx)]">{{ idx + 1 }}</span>
                  <span class="truncate text-sm text-gray-700 dark:text-gray-300">{{ user.email }}</span>
                </div>
                <span class="shrink-0 text-sm font-medium text-gray-900 dark:text-white">&yen;{{ user.amount.toFixed(2) }}</span>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI } from '@/api/admin/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { DashboardStats } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatsCards from '@/components/admin/payment/OrderStatsCards.vue'
import DailyRevenueChart from '@/components/admin/payment/DailyRevenueChart.vue'

const { t } = useI18n()
const appStore = useAppStore()

const DAYS_OPTIONS = [7, 30, 90] as const
const days = ref<number>(30)
const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

const opsSections = computed(() => {
  const ops = stats.value?.ops
  if (!ops) return []

  const danger = 'text-red-600 dark:text-red-300'
  const warning = 'text-amber-600 dark:text-amber-300'
  const success = 'text-green-600 dark:text-green-300'
  const normal = 'text-gray-900 dark:text-white'

  return [
    {
      title: t('payment.admin.ops.callback.title'),
      description: t('payment.admin.ops.callback.desc'),
      icon: 'exclamationTriangle' as const,
      iconBg: 'bg-red-100 dark:bg-red-900/30',
      iconClass: 'text-red-600 dark:text-red-300',
      items: [
        { label: t('payment.admin.ops.callback.failures'), value: ops.callback_failures, className: ops.callback_failures > 0 ? danger : success },
        { label: t('payment.admin.ops.callback.inconsistencies'), value: ops.order_inconsistencies, className: ops.order_inconsistencies > 0 ? danger : success },
        { label: t('payment.admin.ops.callback.fulfillmentFailed'), value: ops.fulfillment_failed, className: ops.fulfillment_failed > 0 ? warning : success },
        { label: t('payment.admin.ops.callback.paidNotCompleted'), value: ops.paid_not_completed, className: ops.paid_not_completed > 0 ? warning : success },
        { label: t('payment.admin.ops.callback.stalePending'), value: ops.stale_pending, className: ops.stale_pending > 0 ? warning : success },
      ],
    },
    {
      title: t('payment.admin.ops.refunds.title'),
      description: t('payment.admin.ops.refunds.desc'),
      icon: 'refresh' as const,
      iconBg: 'bg-amber-100 dark:bg-amber-900/30',
      iconClass: 'text-amber-600 dark:text-amber-300',
      items: [
        { label: t('payment.admin.ops.refunds.requested'), value: ops.refund_requested, className: ops.refund_requested > 0 ? warning : normal },
        { label: t('payment.admin.ops.refunds.processing'), value: ops.refunding, className: ops.refunding > 0 ? warning : normal },
        { label: t('payment.admin.ops.refunds.failed'), value: ops.refund_failed, className: ops.refund_failed > 0 ? danger : success },
        { label: t('payment.admin.ops.refunds.completed'), value: ops.refunded, className: normal },
      ],
    },
    {
      title: t('payment.admin.ops.providers.title'),
      description: t('payment.admin.ops.providers.desc'),
      icon: 'server' as const,
      iconBg: 'bg-blue-100 dark:bg-blue-900/30',
      iconClass: 'text-blue-600 dark:text-blue-300',
      items: [
        { label: t('payment.admin.ops.providers.enabled'), value: ops.enabled_provider_instances, className: success },
        { label: t('payment.admin.ops.providers.disabled'), value: ops.disabled_provider_instances, className: ops.disabled_provider_instances > 0 ? warning : normal },
        { label: t('payment.admin.ops.providers.unavailable'), value: ops.provider_unavailable, className: ops.provider_unavailable > 0 ? warning : normal },
        { label: t('payment.admin.ops.providers.refundEnabled'), value: ops.refund_enabled_provider_instances, className: normal },
        { label: t('payment.admin.ops.providers.userRefundEnabled'), value: ops.user_refund_enabled_provider_instances, className: normal },
      ],
    },
  ]
})

function methodColor(type: string): string {
  const c: Record<string, string> = {
    alipay: 'bg-blue-500', wxpay: 'bg-green-500',
    alipay_direct: 'bg-blue-400', wxpay_direct: 'bg-green-400',
    stripe: 'bg-purple-500',
  }
  return c[type] || 'bg-gray-400'
}

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
  if (idx === 1) return 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
  if (idx === 2) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400'
}

async function loadDashboard() {
  loading.value = true
  try {
    const res = await adminPaymentAPI.getDashboard(days.value)
    stats.value = res.data
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

watch(days, () => loadDashboard())
onMounted(() => loadDashboard())
</script>
