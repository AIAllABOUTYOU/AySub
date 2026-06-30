<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <div
          class="overflow-hidden rounded-lg border border-emerald-200 bg-white shadow-sm dark:border-emerald-900/50 dark:bg-dark-900"
        >
          <div class="bg-gradient-to-br from-emerald-500 to-primary-600 px-6 py-8 sm:px-8">
            <div class="flex flex-col gap-6 sm:flex-row sm:items-center sm:justify-between">
              <div class="min-w-0">
                <div
                  class="mb-4 inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-white/20 text-white backdrop-blur-sm"
                >
                  <Icon name="badge" size="xl" />
                </div>
                <h1 class="text-2xl font-bold text-white sm:text-3xl">
                  {{ t('checkin.title') }}
                </h1>
                <p class="mt-2 max-w-xl text-sm leading-6 text-emerald-50">
                  {{ t('checkin.description') }}
                </p>
              </div>

              <div
                class="rounded-lg border border-white/20 bg-white/15 px-4 py-3 text-left text-white backdrop-blur-sm sm:min-w-40"
              >
                <p class="text-xs font-medium uppercase tracking-wide text-emerald-50">
                  {{ t('checkin.reward') }}
                </p>
                <p class="mt-1 text-2xl font-bold">
                  {{ rewardAmountLabel }}
                </p>
              </div>
            </div>
          </div>

          <div class="space-y-6 p-5 sm:p-6">
            <div
              v-if="errorMessage"
              class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-300"
            >
              {{ errorMessage }}
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.todayStatus') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ checkedInToday ? t('checkin.checkedIn') : t('checkin.notCheckedIn') }}
                </p>
              </div>
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.streak') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('checkin.streakDays', { days: status?.streak_days ?? 0 }) }}
                </p>
              </div>
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.nextAvailable') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ formatDateTime(status?.next_checkin_at) }}
                </p>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-3 sm:grid-cols-4">
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.totalCheckins') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ status?.stats?.total_checkins ?? 0 }}
                </p>
              </div>
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.monthCheckins') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ status?.stats?.checkin_count ?? 0 }}
                </p>
              </div>
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.totalRewards') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ formatAmount(status?.stats?.total_quota) }}
                </p>
              </div>
              <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-800/70">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('checkin.monthRewards') }}
                </p>
                <p class="mt-2 text-base font-semibold text-gray-900 dark:text-white">
                  {{ formatAmount(monthRewardTotal) }}
                </p>
              </div>
            </div>

            <button
              type="button"
              class="btn btn-primary w-full py-3 text-base"
              :disabled="submitting || checkedInToday"
              @click="handleCheckin"
            >
              <svg
                v-if="submitting"
                class="-ml-1 mr-2 h-5 w-5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
              <Icon v-else name="checkCircle" size="md" class="mr-2" />
              {{
                submitting
                  ? t('checkin.submitting')
                  : checkedInToday
                    ? t('checkin.alreadyCheckedIn')
                    : t('checkin.checkinButton')
              }}
            </button>

            <div
              v-if="lastResult"
              class="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-900/20 dark:text-emerald-300"
            >
              <p class="font-medium">{{ lastResult.message || t('checkin.success') }}</p>
              <p v-if="lastResult.new_balance !== undefined" class="mt-1">
                {{ t('checkin.newBalance') }}: {{ formatAmount(lastResult.new_balance) }}
              </p>
            </div>

            <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
              <div class="mb-4 flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('checkin.history') }}
                  </h2>
                  <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                    {{ t('checkin.historyHint') }}
                  </p>
                </div>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-600 transition-colors hover:bg-gray-100 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300 dark:hover:bg-dark-700"
                    @click="changeMonth(-1)"
                  >
                    <Icon name="chevronLeft" size="sm" />
                  </button>
                  <div class="min-w-28 text-center text-sm font-medium text-gray-900 dark:text-white">
                    {{ currentMonthLabel }}
                  </div>
                  <button
                    type="button"
                    class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-600 transition-colors hover:bg-gray-100 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300 dark:hover:bg-dark-700"
                    @click="changeMonth(1)"
                  >
                    <Icon name="chevronRight" size="sm" />
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-7 gap-1">
                <div
                  v-for="weekday in weekDays"
                  :key="weekday"
                  class="flex h-7 items-center justify-center text-xs font-medium text-gray-500 dark:text-dark-400"
                >
                  {{ weekday }}
                </div>
                <div
                  v-for="day in calendarDays"
                  :key="day.key"
                  class="relative flex h-12 flex-col items-center justify-center rounded-lg border text-xs sm:h-14"
                  :class="dayClass(day)"
                >
                  <template v-if="day.inMonth">
                    <span class="font-medium tabular-nums">{{ day.day }}</span>
                    <span
                      v-if="day.reward !== undefined"
                      class="mt-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-300"
                    >
                      +{{ formatCompactAmount(day.reward) }}
                    </span>
                  </template>
                </div>
              </div>

              <div class="mt-5 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
                <div
                  v-if="monthlyRecords.length === 0"
                  class="flex min-h-24 items-center justify-center px-4 py-6 text-sm text-gray-500 dark:text-dark-400"
                >
                  {{ t('checkin.historyEmpty') }}
                </div>
                <div v-else class="divide-y divide-gray-100 dark:divide-dark-800">
                  <div
                    v-for="record in monthlyRecords"
                    :key="record.checkin_date"
                    class="flex items-center justify-between gap-4 px-4 py-3"
                  >
                    <div>
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ formatDate(record.checkin_date) }}
                      </p>
                      <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                        {{ record.checkin_date }}
                      </p>
                    </div>
                    <div class="text-sm font-semibold text-emerald-600 dark:text-emerald-300">
                      +{{ formatAmount(recordReward(record)) }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { checkinAPI } from '@/api/checkin'
import type { CheckinRecord, CheckinResult, CheckinStatus } from '@/api/checkin'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t, locale } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(true)
const submitting = ref(false)
const status = ref<CheckinStatus | null>(null)
const lastResult = ref<CheckinResult | null>(null)
const errorMessage = ref('')
const currentMonth = ref(formatMonth(new Date()))

const rewardAmount = computed(
  () => status.value?.reward_amount ?? appStore.cachedPublicSettings?.checkin_reward_amount ?? 0,
)
const rewardAmountLabel = computed(() => {
  const mode = status.value?.reward_mode ?? appStore.cachedPublicSettings?.checkin_reward_mode
  if (mode !== 'random') return formatAmount(rewardAmount.value)
  const minAmount =
    status.value?.reward_min_amount ?? appStore.cachedPublicSettings?.checkin_reward_min_amount ?? 0
  const maxAmount =
    status.value?.reward_max_amount ?? appStore.cachedPublicSettings?.checkin_reward_max_amount ?? 0
  return `${formatAmount(minAmount)} - ${formatAmount(maxAmount)}`
})
const checkedInToday = computed(() => status.value?.checked_in_today === true)
const monthlyRecords = computed(() => status.value?.stats?.records ?? [])
const monthRewardTotal = computed(() =>
  monthlyRecords.value.reduce((total, record) => total + recordReward(record), 0),
)
const currentMonthLabel = computed(() => {
  const date = parseMonth(currentMonth.value)
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
  }).format(date)
})
const weekDays = computed(() => {
  const base = new Date(2024, 0, 7)
  return Array.from({ length: 7 }, (_, index) =>
    new Intl.DateTimeFormat(locale.value, { weekday: 'short' }).format(
      new Date(base.getFullYear(), base.getMonth(), base.getDate() + index),
    ),
  )
})
const recordMap = computed(() => {
  const map = new Map<string, number>()
  for (const record of monthlyRecords.value) {
    map.set(record.checkin_date, recordReward(record))
  }
  return map
})
const calendarDays = computed(() => {
  const monthDate = parseMonth(currentMonth.value)
  const year = monthDate.getFullYear()
  const month = monthDate.getMonth()
  const firstDay = new Date(year, month, 1)
  const totalDays = new Date(year, month + 1, 0).getDate()
  const leading = firstDay.getDay()
  const days: Array<{
    key: string
    day: number
    date: string
    inMonth: boolean
    isToday: boolean
    reward?: number
  }> = []
  for (let index = 0; index < leading; index += 1) {
    days.push({
      key: `empty-${index}`,
      day: 0,
      date: '',
      inMonth: false,
      isToday: false,
    })
  }
  const today = formatDateKey(new Date())
  for (let day = 1; day <= totalDays; day += 1) {
    const date = formatDateKey(new Date(year, month, day))
    days.push({
      key: date,
      day,
      date,
      inMonth: true,
      isToday: date === today,
      reward: recordMap.value.get(date),
    })
  }
  while (days.length % 7 !== 0) {
    days.push({
      key: `tail-${days.length}`,
      day: 0,
      date: '',
      inMonth: false,
      isToday: false,
    })
  }
  return days
})

function formatAmount(value: number | undefined): string {
  const amount = Number(value) || 0
  return `$${amount.toFixed(2)}`
}

function formatCompactAmount(value: number | undefined): string {
  const amount = Number(value) || 0
  return amount.toFixed(amount >= 10 ? 0 : 2)
}

function formatDateTime(value?: string | null): string {
  if (!value) return t('checkin.tomorrow')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return t('checkin.tomorrow')
  return new Intl.DateTimeFormat(locale.value, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function formatDate(value: string): string {
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(year, month - 1, day)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    month: 'short',
    day: 'numeric',
    weekday: 'short',
  }).format(date)
}

function formatDateKey(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(
    date.getDate(),
  ).padStart(2, '0')}`
}

function formatMonth(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

function parseMonth(value: string): Date {
  const [year, month] = value.split('-').map(Number)
  return new Date(year, month - 1, 1)
}

function recordReward(record: CheckinRecord): number {
  return Number(record.reward_amount ?? record.quota_awarded) || 0
}

function dayClass(day: { inMonth: boolean; isToday: boolean; reward?: number }): string {
  if (!day.inMonth) {
    return 'border-transparent bg-transparent'
  }
  if (day.reward !== undefined) {
    return day.isToday
      ? 'border-emerald-500 bg-emerald-600 text-white shadow-sm [&_span:last-child]:text-white'
      : 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-900/20 dark:text-emerald-200'
  }
  if (day.isToday) {
    return 'border-primary-300 bg-primary-50 text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300'
  }
  return 'border-gray-200 bg-white text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-300'
}

async function loadStatus(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    await appStore.fetchPublicSettings()
    status.value = await checkinAPI.getStatus(currentMonth.value)
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, t('checkin.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function handleCheckin(): Promise<void> {
  if (checkedInToday.value || submitting.value) return
  submitting.value = true
  errorMessage.value = ''
  try {
    const result = await checkinAPI.checkin()
    status.value = result
    lastResult.value = result
    await authStore.refreshUser()
    await loadStatus()
    appStore.showSuccess(result.message || t('checkin.success'))
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, t('checkin.failed'))
    appStore.showError(errorMessage.value)
  } finally {
    submitting.value = false
  }
}

function changeMonth(offset: number): void {
  const date = parseMonth(currentMonth.value)
  currentMonth.value = formatMonth(new Date(date.getFullYear(), date.getMonth() + offset, 1))
  loadStatus()
}

onMounted(() => {
  loadStatus()
})
</script>
