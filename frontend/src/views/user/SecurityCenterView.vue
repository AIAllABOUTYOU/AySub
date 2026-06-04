<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 p-5 dark:border-dark-700 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div class="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="shield" size="lg" />
              </div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
                {{ t('securityCenter.user.title') }}
              </h1>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                {{ t('securityCenter.user.description') }}
              </p>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:min-w-64">
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('securityCenter.stats.total') }}</div>
                <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ pagination.total.toLocaleString() }}</div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('securityCenter.stats.pageHighRisk') }}</div>
                <div class="mt-1 text-xl font-semibold text-amber-600 dark:text-amber-400">{{ pageElevatedRiskCount }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-3 p-5 sm:grid-cols-3 sm:p-6">
          <RouterLink
            to="/profile"
            class="rounded-lg border border-gray-200 p-4 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:bg-primary-900/20"
          >
            <div class="flex items-center gap-3">
              <Icon name="lock" size="md" class="text-primary-600 dark:text-primary-300" />
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ t('securityCenter.actions.password') }}</div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('securityCenter.actions.profileLinkHint') }}</div>
              </div>
            </div>
          </RouterLink>
          <RouterLink
            to="/profile"
            class="rounded-lg border border-gray-200 p-4 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:bg-primary-900/20"
          >
            <div class="flex items-center gap-3">
              <Icon name="key" size="md" class="text-primary-600 dark:text-primary-300" />
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ t('securityCenter.actions.totp') }}</div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('securityCenter.actions.profileLinkHint') }}</div>
              </div>
            </div>
          </RouterLink>
          <RouterLink
            to="/profile"
            class="rounded-lg border border-gray-200 p-4 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:hover:border-primary-700 dark:hover:bg-primary-900/20"
          >
            <div class="flex items-center gap-3">
              <Icon name="link" size="md" class="text-primary-600 dark:text-primary-300" />
              <div>
                <div class="font-medium text-gray-900 dark:text-white">{{ t('securityCenter.actions.bindings') }}</div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('securityCenter.actions.profileLinkHint') }}</div>
              </div>
            </div>
          </RouterLink>
        </div>
      </section>

      <section class="card space-y-4 p-4 sm:p-5">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
          <div>
            <label class="input-label">{{ t('securityCenter.filters.result') }}</label>
            <Select v-model="filters.result" :options="resultOptions" @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.riskLevel') }}</label>
            <Select v-model="filters.risk_level" :options="riskOptions" @change="applyFilters" />
          </div>
          <div class="lg:col-span-2">
            <label class="input-label">{{ t('securityCenter.filters.action') }}</label>
            <input v-model.trim="filters.action" class="input" :placeholder="t('securityCenter.filters.actionPlaceholder')" @keyup.enter="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('common.startDate') }}</label>
            <input v-model="filters.start_time" type="date" class="input" @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('common.endDate') }}</label>
            <input v-model="filters.end_time" type="date" class="input" @change="applyFilters" />
          </div>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div class="flex-1">
            <label class="input-label">{{ t('common.search') }}</label>
            <input v-model.trim="filters.q" class="input" :placeholder="t('securityCenter.filters.userSearchPlaceholder')" @keyup.enter="applyFilters" />
          </div>
          <div class="flex gap-2">
            <button class="btn btn-secondary flex-1 sm:flex-none" :disabled="loading" @click="loadEvents">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
            <button class="btn btn-secondary flex-1 sm:flex-none" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
          </div>
        </div>
      </section>

      <section class="card space-y-4 p-4 sm:p-5">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('securityCenter.apiKeys.title') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('securityCenter.apiKeys.description') }}</p>
          </div>
          <button class="btn btn-secondary" :disabled="keysLoading" @click="loadAPIKeys">
            <Icon name="refresh" size="sm" :class="keysLoading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </div>
        <div v-if="keysLoading" class="flex justify-center py-6"><LoadingSpinner /></div>
        <div v-else-if="apiKeys.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('securityCenter.apiKeys.empty') }}</div>
        <div v-else class="grid gap-3 md:grid-cols-2">
          <article v-for="key in apiKeys" :key="key.id" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate font-medium text-gray-900 dark:text-white">{{ key.name }}</div>
                <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ key.id }} · {{ key.status }}</div>
              </div>
              <button class="btn btn-danger btn-sm" :disabled="revokingKeyId === key.id" @click="openRevokeDialog(key)">
                {{ revokingKeyId === key.id ? t('common.processing') : t('securityCenter.apiKeys.revoke') }}
              </button>
            </div>
          </article>
        </div>
      </section>

      <section class="space-y-3">
        <div v-if="loading" class="card flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>

        <div v-else-if="events.length === 0" class="card p-8 text-center">
          <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <Icon name="shield" size="lg" />
          </div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('securityCenter.empty.title') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('securityCenter.empty.description') }}</p>
        </div>

        <template v-else>
          <article
            v-for="event in events"
            :key="event.id"
            class="card p-4 sm:p-5"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="font-medium text-gray-900 dark:text-white">{{ displayAction(event) }}</span>
                  <span :class="resultBadgeClass(event.result)">{{ resultLabel(event.result) }}</span>
                  <span :class="riskBadgeClass(event.risk_level)">{{ riskLabel(event.risk_level) }}</span>
                </div>
                <p v-if="event.reason" class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ event.reason }}</p>
                <div class="mt-3 grid gap-2 text-xs text-gray-500 dark:text-gray-400 sm:grid-cols-2">
                  <div>{{ t('securityCenter.fields.time') }}: {{ formatDateTime(event.created_at) || '-' }}</div>
                  <div>{{ t('securityCenter.fields.ip') }}: {{ event.ip || '-' }}</div>
                  <div>{{ t('securityCenter.fields.endpoint') }}: {{ event.endpoint || '-' }}</div>
                  <div>{{ t('securityCenter.fields.requestId') }}: {{ event.request_id || '-' }}</div>
                </div>
              </div>
              <div class="text-xs text-gray-400 dark:text-gray-500">#{{ event.id }}</div>
            </div>
          </article>
        </template>

        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </section>

      <div v-if="showSensitiveVerifyDialog" class="fixed inset-0 z-50 overflow-y-auto" @click.self="closeRevokeDialog">
        <div class="flex min-h-full items-center justify-center p-4">
          <div class="fixed inset-0 bg-black/50 transition-opacity" @click="closeRevokeDialog"></div>
          <div class="relative w-full max-w-md transform rounded-xl bg-white p-6 shadow-xl transition-all dark:bg-dark-800">
            <div class="mb-5">
              <h3 class="text-xl font-semibold text-gray-900 dark:text-white">
                {{ t('securityCenter.sensitiveVerify.title') }}
              </h3>
              <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                {{ t('securityCenter.sensitiveVerify.description', { name: selectedAPIKey?.name || '-' }) }}
              </p>
            </div>

            <form class="space-y-4" @submit.prevent="verifyAndRevokeAPIKey">
              <div>
                <label class="input-label">{{ t('securityCenter.sensitiveVerify.method') }}</label>
                <Select v-model="sensitiveVerifyMode" :options="sensitiveVerifyOptions" />
              </div>
              <div v-if="sensitiveVerifyMode === 'password'">
                <label class="input-label">{{ t('common.password') }}</label>
                <input
                  v-model="sensitiveVerifyForm.password"
                  type="password"
                  autocomplete="current-password"
                  class="input"
                  :disabled="sensitiveVerifying || revokingKeyId !== null"
                  :placeholder="t('securityCenter.sensitiveVerify.passwordPlaceholder')"
                />
              </div>
              <div v-else-if="sensitiveVerifyMode === 'totp'">
                <label class="input-label">{{ t('securityCenter.sensitiveVerify.totpCode') }}</label>
                <input
                  v-model.trim="sensitiveVerifyForm.totpCode"
                  type="text"
                  maxlength="6"
                  inputmode="numeric"
                  pattern="[0-9]*"
                  autocomplete="one-time-code"
                  class="input font-mono"
                  :disabled="sensitiveVerifying || revokingKeyId !== null"
                  :placeholder="t('securityCenter.sensitiveVerify.totpPlaceholder')"
                />
              </div>
              <div v-else>
                <label class="input-label">{{ t('securityCenter.sensitiveVerify.recoveryCode') }}</label>
                <input
                  v-model.trim="sensitiveVerifyForm.recoveryCode"
                  type="text"
                  autocomplete="one-time-code"
                  class="input font-mono uppercase"
                  :disabled="sensitiveVerifying || revokingKeyId !== null"
                  :placeholder="t('securityCenter.sensitiveVerify.recoveryPlaceholder')"
                />
              </div>
              <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-200">
                {{ t('securityCenter.sensitiveVerify.revokeWarning') }}
              </div>
              <div class="flex justify-end gap-3 pt-2">
                <button type="button" class="btn btn-secondary" :disabled="sensitiveVerifying || revokingKeyId !== null" @click="closeRevokeDialog">
                  {{ t('common.cancel') }}
                </button>
                <button
                  type="submit"
                  class="btn btn-danger"
                  :disabled="!canSubmitSensitiveVerify || sensitiveVerifying || revokingKeyId !== null"
                >
                  {{ sensitiveVerifying || revokingKeyId !== null ? t('common.processing') : t('securityCenter.apiKeys.revoke') }}
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import { securityAPI, type SecurityAuditLog, type UserSecurityEventFilterParams } from '@/api/security'
import { keysAPI } from '@/api/keys'
import type { ApiKey } from '@/types'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const events = ref<SecurityAuditLog[]>([])
const apiKeys = ref<ApiKey[]>([])
const loading = ref(false)
const keysLoading = ref(false)
const revokingKeyId = ref<number | null>(null)
const showSensitiveVerifyDialog = ref(false)
const selectedAPIKey = ref<ApiKey | null>(null)
const sensitiveVerifyMode = ref<'password' | 'totp' | 'recovery_code'>('password')
const sensitiveVerifying = ref(false)
const sensitiveVerifyForm = reactive({
  password: '',
  totpCode: '',
  recoveryCode: '',
})
let abortController: AbortController | null = null

const filters = reactive({
  action: '',
  result: '',
  risk_level: '',
  q: '',
  start_time: '',
  end_time: '',
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0,
})

const resultOptions = computed(() => [
  { value: '', label: t('securityCenter.result.all') },
  { value: 'success', label: t('securityCenter.result.success') },
  { value: 'denied', label: t('securityCenter.result.denied') },
  { value: 'failure', label: t('securityCenter.result.failure') },
])

const riskOptions = computed(() => [
  { value: '', label: t('securityCenter.risk.all') },
  { value: 'low', label: t('securityCenter.risk.low') },
  { value: 'medium', label: t('securityCenter.risk.medium') },
  { value: 'high', label: t('securityCenter.risk.high') },
  { value: 'critical', label: t('securityCenter.risk.critical') },
])

const sensitiveVerifyOptions = computed(() => [
  { value: 'password', label: t('securityCenter.sensitiveVerify.password') },
  { value: 'totp', label: t('securityCenter.sensitiveVerify.totp') },
  { value: 'recovery_code', label: t('securityCenter.sensitiveVerify.recoveryCode') },
])

const canSubmitSensitiveVerify = computed(() => {
  if (!selectedAPIKey.value) return false
  if (sensitiveVerifyMode.value === 'password') return sensitiveVerifyForm.password.length > 0
  if (sensitiveVerifyMode.value === 'totp') return sensitiveVerifyForm.totpCode.length === 6
  return sensitiveVerifyForm.recoveryCode.length >= 10
})

const pageElevatedRiskCount = computed(() =>
  events.value.filter((event) => ['high', 'critical'].includes(String(event.risk_level))).length,
)

function toDayStartIso(value: string): string {
  return new Date(`${value}T00:00:00`).toISOString()
}

function toDayEndIso(value: string): string {
  return new Date(`${value}T23:59:59.999`).toISOString()
}

function buildParams(): UserSecurityEventFilterParams {
  const params: UserSecurityEventFilterParams = {
    page: pagination.page,
    page_size: pagination.page_size,
  }
  if (filters.action) params.action = filters.action
  if (filters.result) params.result = filters.result
  if (filters.risk_level) params.risk_level = filters.risk_level
  if (filters.q) params.q = filters.q
  if (filters.start_time) params.start_time = toDayStartIso(filters.start_time)
  if (filters.end_time) params.end_time = toDayEndIso(filters.end_time)
  return params
}

async function loadEvents(): Promise<void> {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await securityAPI.listUserEvents(buildParams(), { signal: controller.signal })
    if (controller.signal.aborted) return
    events.value = res.items || []
    pagination.total = res.total || 0
    pagination.page = res.page || pagination.page
    pagination.page_size = res.page_size || pagination.page_size
    pagination.pages = res.pages || 0
  } catch (error) {
    if (controller.signal.aborted) return
    events.value = []
    pagination.total = 0
    pagination.pages = 0
    appStore.showError(extractApiErrorMessage(error, t('securityCenter.failedToLoad')))
  } finally {
    if (abortController === controller) loading.value = false
  }
}

async function loadAPIKeys(): Promise<void> {
  keysLoading.value = true
  try {
    const res = await keysAPI.list(1, 6, { sort_by: 'last_used_at', sort_order: 'desc' })
    apiKeys.value = res.items || []
  } catch (error) {
    apiKeys.value = []
    appStore.showError(extractApiErrorMessage(error, t('securityCenter.apiKeys.loadFailed')))
  } finally {
    keysLoading.value = false
  }
}

function openRevokeDialog(key: ApiKey): void {
  selectedAPIKey.value = key
  sensitiveVerifyMode.value = 'password'
  sensitiveVerifyForm.password = ''
  sensitiveVerifyForm.totpCode = ''
  sensitiveVerifyForm.recoveryCode = ''
  showSensitiveVerifyDialog.value = true
}

function closeRevokeDialog(): void {
  if (sensitiveVerifying.value || revokingKeyId.value !== null) return
  showSensitiveVerifyDialog.value = false
  selectedAPIKey.value = null
  sensitiveVerifyForm.password = ''
  sensitiveVerifyForm.totpCode = ''
  sensitiveVerifyForm.recoveryCode = ''
}

async function verifyAndRevokeAPIKey(): Promise<void> {
  const key = selectedAPIKey.value
  if (!key || !canSubmitSensitiveVerify.value) return
  sensitiveVerifying.value = true
  let sensitiveToken = ''
  try {
    const verifyPayload: {
      action: string
      password?: string
      totp_code?: string
      recovery_code?: string
    } = { action: 'api_key.revoke' }
    if (sensitiveVerifyMode.value === 'password') verifyPayload.password = sensitiveVerifyForm.password
    if (sensitiveVerifyMode.value === 'totp') verifyPayload.totp_code = sensitiveVerifyForm.totpCode
    if (sensitiveVerifyMode.value === 'recovery_code') verifyPayload.recovery_code = sensitiveVerifyForm.recoveryCode
    const verified = await securityAPI.verifySensitiveOperation(verifyPayload)
    sensitiveToken = verified.token
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('securityCenter.sensitiveVerify.failed')))
    sensitiveVerifying.value = false
    return
  }
  sensitiveVerifying.value = false
  await revokeAPIKey(key.id, sensitiveToken)
}

async function revokeAPIKey(id: number, sensitiveToken: string): Promise<void> {
  revokingKeyId.value = id
  try {
    await securityAPI.revokeUserAPIKey(id, sensitiveToken)
    appStore.showSuccess(t('securityCenter.apiKeys.revoked'))
    showSensitiveVerifyDialog.value = false
    selectedAPIKey.value = null
    await Promise.all([loadAPIKeys(), loadEvents()])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('securityCenter.apiKeys.revokeFailed')))
  } finally {
    revokingKeyId.value = null
  }
}

function applyFilters(): void {
  pagination.page = 1
  loadEvents()
}

function resetFilters(): void {
  filters.action = ''
  filters.result = ''
  filters.risk_level = ''
  filters.q = ''
  filters.start_time = ''
  filters.end_time = ''
  pagination.page = 1
  loadEvents()
}

function handlePageChange(page: number): void {
  pagination.page = page
  loadEvents()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  loadEvents()
}

function displayAction(event: SecurityAuditLog): string {
  return event.action || event.event_type || t('securityCenter.fields.unknownAction')
}

function resultLabel(value: string): string {
  const key = `securityCenter.result.${value}`
  const label = t(key)
  return label === key ? value || '-' : label
}

function riskLabel(value: string): string {
  const key = `securityCenter.risk.${value}`
  const label = t(key)
  return label === key ? value || '-' : label
}

function resultBadgeClass(value: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (value === 'success') return `${base} bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  if (value === 'denied') return `${base} bg-amber-50 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
  if (value === 'failure') return `${base} bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300`
  return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
}

function riskBadgeClass(value: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (value === 'critical') return `${base} bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200`
  if (value === 'high') return `${base} bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-200`
  if (value === 'medium') return `${base} bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-200`
  return `${base} bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300`
}

onMounted(() => {
  loadEvents()
  loadAPIKeys()
})
</script>
