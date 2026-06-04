<template>
  <AppLayout>
    <div class="mx-auto max-w-3xl space-y-6">
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
                  {{ formatAmount(rewardAmount) }}
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
import type { CheckinResult, CheckinStatus } from '@/api/checkin'
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

const rewardAmount = computed(
  () => status.value?.reward_amount ?? appStore.cachedPublicSettings?.checkin_reward_amount ?? 0,
)
const checkedInToday = computed(() => status.value?.checked_in_today === true)

function formatAmount(value: number | undefined): string {
  const amount = Number(value) || 0
  return `$${amount.toFixed(2)}`
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

async function loadStatus(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    await appStore.fetchPublicSettings()
    status.value = await checkinAPI.getStatus()
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
    appStore.showSuccess(result.message || t('checkin.success'))
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, t('checkin.failed'))
    appStore.showError(errorMessage.value)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadStatus()
})
</script>
