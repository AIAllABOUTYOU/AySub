<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <section class="grid gap-4 lg:grid-cols-[1fr_360px]">
        <div class="card p-5 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div class="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="shield" size="lg" />
              </div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
                {{ t('admin.securityCenter.title') }}
              </h1>
              <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                {{ t('admin.securityCenter.description') }}
              </p>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:min-w-72">
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

        <div class="card p-5 sm:p-6">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.securityCenter.integrity.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.integrity.description') }}</p>
            </div>
            <span v-if="integrity" :class="integrityBadgeClass">
              {{ integrity.valid ? t('admin.securityCenter.integrity.valid') : t('admin.securityCenter.integrity.invalid') }}
            </span>
          </div>

          <div v-if="integrity" class="mt-4 space-y-2 text-sm text-gray-600 dark:text-gray-300">
            <div>{{ t('admin.securityCenter.integrity.checked') }}: {{ integrity.checked.toLocaleString() }}</div>
            <div v-if="integrity.broken_at_id" class="text-red-600 dark:text-red-300">
              {{ t('admin.securityCenter.integrity.brokenAt') }}: #{{ integrity.broken_at_id }}
            </div>
          </div>
          <p v-else class="mt-4 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.securityCenter.integrity.notChecked') }}
          </p>

          <button class="btn btn-primary mt-5 w-full" :disabled="integrityLoading" @click="runIntegrityCheck">
            <Icon name="refresh" size="sm" :class="integrityLoading ? 'animate-spin' : ''" />
            {{ integrityLoading ? t('admin.securityCenter.integrity.checking') : t('admin.securityCenter.integrity.check') }}
          </button>
        </div>
      </section>

      <section class="grid gap-4 xl:grid-cols-4">
        <div class="card p-4 sm:p-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.securityCenter.incidents.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.incidents.description') }}</p>
            </div>
            <button class="btn btn-secondary btn-sm" :disabled="entitiesLoading" @click="loadSecurityEntities">
              <Icon name="refresh" size="sm" :class="entitiesLoading ? 'animate-spin' : ''" />
            </button>
          </div>
          <div class="mt-4 space-y-3">
            <div v-if="entitiesLoading" class="flex justify-center py-6"><LoadingSpinner /></div>
            <p v-else-if="incidents.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.incidents.empty') }}</p>
            <article v-for="incident in incidents" v-else :key="incident.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex items-center justify-between gap-2">
                <div class="min-w-0 truncate font-medium text-gray-900 dark:text-white">{{ incident.title || incident.incident_key }}</div>
                <span :class="riskBadgeClass(incident.severity)">{{ riskLabel(incident.severity) }}</span>
              </div>
              <div class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ incident.status }} · {{ formatPrincipal(incident.subject_type, incident.subject_id, '') }}
              </div>
              <div class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ formatDateTime(incident.detected_at) || '-' }}</div>
            </article>
          </div>
        </div>

        <div class="card p-4 sm:p-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.securityCenter.policies.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.policies.description') }}</p>
            </div>
            <span class="text-xs text-gray-400 dark:text-gray-500">{{ policies.length }}</span>
          </div>
          <div class="mt-4 grid gap-3">
            <input v-model.trim="policyForm.name" class="input" :placeholder="t('admin.securityCenter.policies.name')" />
            <input v-model.trim="policyForm.code" class="input" :placeholder="t('admin.securityCenter.policies.code')" />
            <div class="grid gap-3 sm:grid-cols-2">
              <Select v-model="policyForm.action" :options="policyActionOptions" />
              <Select v-model="policyForm.severity" :options="riskOptionsNoAll" />
            </div>
            <textarea v-model.trim="policyForm.description" class="input min-h-20" :placeholder="t('admin.securityCenter.policies.descriptionField')" />
            <textarea v-model.trim="policyForm.conditionsText" class="input min-h-24 font-mono text-xs" placeholder='{"endpoint":"POST /v1/chat/completions","model":"gpt-*"}' />
            <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="policyForm.enabled" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
              {{ t('admin.securityCenter.policies.enabled') }}
            </label>
            <div class="flex gap-2">
              <button class="btn btn-primary flex-1" :disabled="policySaving" @click="savePolicy">
                {{ policyForm.id ? t('common.save') : t('common.create') }}
              </button>
              <button class="btn btn-secondary" @click="resetPolicyForm">{{ t('common.reset') }}</button>
            </div>
          </div>
          <div class="mt-5 space-y-2">
            <p v-if="policies.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.policies.empty') }}</p>
            <article v-for="policy in policies" :key="policy.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex items-center justify-between gap-2">
                <div class="min-w-0">
                  <div class="truncate font-medium text-gray-900 dark:text-white">{{ policy.name }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ policy.code }} · {{ policy.action }}</div>
                </div>
                <span :class="policy.enabled ? resultBadgeClass('success') : resultBadgeClass('failure')">
                  {{ policy.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
              </div>
              <div class="mt-3 flex gap-2">
                <button class="btn btn-secondary btn-sm flex-1" @click="editPolicy(policy)">{{ t('common.edit') }}</button>
                <button class="btn btn-danger btn-sm flex-1" @click="deletePolicy(policy)">{{ t('common.delete') }}</button>
              </div>
            </article>
          </div>
        </div>

        <div class="card p-4 sm:p-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.securityCenter.locks.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.locks.description') }}</p>
            </div>
            <span class="text-xs text-gray-400 dark:text-gray-500">{{ locks.length }}</span>
          </div>
          <div class="mt-4 grid gap-3">
            <Select v-model="lockForm.subject_type" :options="lockSubjectOptions" />
            <input v-model.number="lockForm.subject_id" class="input" type="number" min="1" :placeholder="t('admin.securityCenter.locks.subjectId')" />
            <input v-model.trim="lockForm.reason" class="input" :placeholder="t('admin.securityCenter.locks.reason')" />
            <button class="btn btn-primary" :disabled="lockSaving" @click="createLock">{{ t('admin.securityCenter.locks.create') }}</button>
          </div>
          <div class="mt-5 space-y-2">
            <p v-if="locks.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.locks.empty') }}</p>
            <article v-for="lock in locks" :key="lock.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex items-center justify-between gap-2">
                <div class="font-medium text-gray-900 dark:text-white">{{ formatPrincipal(lock.subject_type, lock.subject_id, '') }}</div>
                <span :class="lock.status === 'active' ? resultBadgeClass('denied') : resultBadgeClass('success')">{{ lock.status }}</span>
              </div>
              <div class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ lock.reason || '-' }}</div>
              <div class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ formatDateTime(lock.locked_at) || '-' }}</div>
              <button v-if="lock.status === 'active'" class="btn btn-secondary btn-sm mt-3 w-full" @click="unlockLock(lock)">
                {{ t('admin.securityCenter.locks.unlock') }}
              </button>
            </article>
          </div>
        </div>

        <div class="card p-4 sm:p-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('admin.securityCenter.exports.title') }}</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.exports.description') }}</p>
            </div>
            <button class="btn btn-secondary btn-sm" :disabled="entitiesLoading" @click="loadSecurityEntities">
              <Icon name="refresh" size="sm" :class="entitiesLoading ? 'animate-spin' : ''" />
            </button>
          </div>

          <button class="btn btn-primary mt-4 w-full" :disabled="exportCreating" @click="createExport">
            <Icon name="download" size="sm" :class="exportCreating ? 'animate-pulse' : ''" />
            {{ exportCreating ? t('admin.securityCenter.exports.creating') : t('admin.securityCenter.exports.create') }}
          </button>

          <div class="mt-5 space-y-2">
            <p v-if="exports.length === 0" class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.securityCenter.exports.empty') }}</p>
            <article v-for="item in exports" :key="item.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
              <div class="flex items-center justify-between gap-2">
                <div class="min-w-0">
                  <div class="truncate font-medium text-gray-900 dark:text-white">#{{ item.id }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ item.row_count.toLocaleString() }} {{ t('admin.securityCenter.exports.rows') }}
                  </div>
                </div>
                <span :class="exportStatusBadgeClass(item.status)">{{ exportStatusLabel(item.status) }}</span>
              </div>
              <div class="mt-2 text-xs text-gray-400 dark:text-gray-500">{{ formatDateTime(item.requested_at) || '-' }}</div>
              <div v-if="item.file_sha256" class="mt-1 truncate text-xs text-gray-400 dark:text-gray-500">SHA256 {{ item.file_sha256 }}</div>
              <div v-if="item.error_message" class="mt-2 text-xs text-red-600 dark:text-red-300">{{ item.error_message }}</div>
              <button
                v-if="item.download_available"
                class="btn btn-secondary btn-sm mt-3 w-full"
                :disabled="downloadingExportId === item.id"
                @click="downloadExport(item)"
              >
                {{ downloadingExportId === item.id ? t('admin.securityCenter.exports.downloading') : t('admin.securityCenter.exports.download') }}
              </button>
            </article>
          </div>
        </div>
      </section>

      <section class="card space-y-4 p-4 sm:p-5">
        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div>
            <label class="input-label">{{ t('securityCenter.filters.result') }}</label>
            <Select v-model="filters.result" :options="resultOptions" @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.riskLevel') }}</label>
            <Select v-model="filters.risk_level" :options="riskOptions" @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.actorType') }}</label>
            <input v-model.trim="filters.actor_type" class="input" placeholder="user / admin / system" @keyup.enter="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.actorId') }}</label>
            <input v-model.number="filters.actor_id" class="input" type="number" min="1" placeholder="ID" @keyup.enter="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.subjectType') }}</label>
            <input v-model.trim="filters.subject_type" class="input" placeholder="user / api_key" @keyup.enter="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('securityCenter.filters.subjectId') }}</label>
            <input v-model.number="filters.subject_id" class="input" type="number" min="1" placeholder="ID" @keyup.enter="applyFilters" />
          </div>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div class="xl:col-span-2">
            <label class="input-label">{{ t('securityCenter.filters.action') }}</label>
            <input v-model.trim="filters.action" class="input" :placeholder="t('securityCenter.filters.actionPlaceholder')" @keyup.enter="applyFilters" />
          </div>
          <div class="xl:col-span-2">
            <label class="input-label">{{ t('securityCenter.fields.requestId') }}</label>
            <input v-model.trim="filters.request_id" class="input" placeholder="req_..." @keyup.enter="applyFilters" />
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
            <input v-model.trim="filters.q" class="input" :placeholder="t('securityCenter.filters.adminSearchPlaceholder')" @keyup.enter="applyFilters" />
          </div>
          <div class="flex gap-2">
            <button class="btn btn-secondary flex-1 sm:flex-none" :disabled="loading" @click="loadLogs">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
            <button class="btn btn-secondary flex-1 sm:flex-none" @click="resetFilters">
              {{ t('common.reset') }}
            </button>
            <button class="btn btn-primary flex-1 sm:flex-none" :disabled="exportCreating" @click="createExport">
              <Icon name="download" size="sm" />
              {{ t('admin.securityCenter.exports.createShort') }}
            </button>
          </div>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div v-if="loading" class="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>

        <div v-else-if="logs.length === 0" class="p-8 text-center">
          <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <Icon name="clipboard" size="lg" />
          </div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('securityCenter.empty.title') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('securityCenter.empty.description') }}</p>
        </div>

        <template v-else>
          <div class="hidden overflow-x-auto lg:block">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.time') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.action') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.actor') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.subject') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.result') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.risk') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('securityCenter.fields.client') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-700 dark:bg-dark-900">
                <tr v-for="log in logs" :key="log.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/70">
                  <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(log.created_at) || '-' }}</td>
                  <td class="max-w-xs px-4 py-3">
                    <div class="font-medium text-gray-900 dark:text-white">{{ displayAction(log) }}</div>
                    <div v-if="log.reason" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ log.reason }}</div>
                    <div class="mt-1 truncate text-xs text-gray-400 dark:text-gray-500">{{ log.endpoint || '-' }}</div>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatPrincipal(log.actor_type, log.actor_id, log.actor_label) }}</td>
                  <td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ formatPrincipal(log.subject_type, log.subject_id, log.subject_label) }}</td>
                  <td class="px-4 py-3"><span :class="resultBadgeClass(log.result)">{{ resultLabel(log.result) }}</span></td>
                  <td class="px-4 py-3"><span :class="riskBadgeClass(log.risk_level)">{{ riskLabel(log.risk_level) }}</span></td>
                  <td class="max-w-[220px] px-4 py-3 text-xs text-gray-500 dark:text-gray-400">
                    <div class="truncate">{{ log.ip || '-' }}</div>
                    <div class="mt-1 truncate">{{ log.request_id || '-' }}</div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="divide-y divide-gray-100 dark:divide-dark-700 lg:hidden">
            <article v-for="log in logs" :key="log.id" class="p-4">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="font-medium text-gray-900 dark:text-white">{{ displayAction(log) }}</span>
                    <span :class="resultBadgeClass(log.result)">{{ resultLabel(log.result) }}</span>
                    <span :class="riskBadgeClass(log.risk_level)">{{ riskLabel(log.risk_level) }}</span>
                  </div>
                  <p v-if="log.reason" class="mt-2 text-sm text-gray-600 dark:text-gray-400">{{ log.reason }}</p>
                </div>
                <span class="text-xs text-gray-400 dark:text-gray-500">#{{ log.id }}</span>
              </div>
              <div class="mt-3 grid gap-2 text-xs text-gray-500 dark:text-gray-400">
                <div>{{ t('securityCenter.fields.time') }}: {{ formatDateTime(log.created_at) || '-' }}</div>
                <div>{{ t('securityCenter.fields.actor') }}: {{ formatPrincipal(log.actor_type, log.actor_id, log.actor_label) }}</div>
                <div>{{ t('securityCenter.fields.subject') }}: {{ formatPrincipal(log.subject_type, log.subject_id, log.subject_label) }}</div>
                <div>{{ t('securityCenter.fields.ip') }}: {{ log.ip || '-' }}</div>
                <div>{{ t('securityCenter.fields.endpoint') }}: {{ log.endpoint || '-' }}</div>
                <div>{{ t('securityCenter.fields.requestId') }}: {{ log.request_id || '-' }}</div>
              </div>
            </article>
          </div>
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
import {
  securityAPI,
  type AdminSecurityAuditFilterParams,
  type SecurityAuditIntegrityResult,
  type SecurityAuditExport,
  type SecurityIncident,
  type SecurityAuditLog,
  type SecurityPolicyRule,
  type SecurityPolicyRuleInput,
  type SecuritySubjectLock,
} from '@/api/security'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const logs = ref<SecurityAuditLog[]>([])
const incidents = ref<SecurityIncident[]>([])
const policies = ref<SecurityPolicyRule[]>([])
const locks = ref<SecuritySubjectLock[]>([])
const exports = ref<SecurityAuditExport[]>([])
const loading = ref(false)
const entitiesLoading = ref(false)
const integrityLoading = ref(false)
const policySaving = ref(false)
const lockSaving = ref(false)
const exportCreating = ref(false)
const downloadingExportId = ref<number | null>(null)
const integrity = ref<SecurityAuditIntegrityResult | null>(null)
let abortController: AbortController | null = null

const filters = reactive({
  action: '',
  result: '',
  risk_level: '',
  actor_type: '',
  actor_id: null as number | null,
  subject_type: '',
  subject_id: null as number | null,
  request_id: '',
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

const policyForm = reactive({
  id: null as number | null,
  name: '',
  code: '',
  description: '',
  enabled: true,
  severity: 'medium',
  action: 'observe',
  conditionsText: '{}',
})

const lockForm = reactive({
  subject_type: 'api_key',
  subject_id: null as number | null,
  reason: '',
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

const riskOptionsNoAll = computed(() => [
  { value: 'low', label: t('securityCenter.risk.low') },
  { value: 'medium', label: t('securityCenter.risk.medium') },
  { value: 'high', label: t('securityCenter.risk.high') },
  { value: 'critical', label: t('securityCenter.risk.critical') },
])

const policyActionOptions = computed(() => [
  { value: 'observe', label: t('admin.securityCenter.policyActions.observe') },
  { value: 'block', label: t('admin.securityCenter.policyActions.block') },
  { value: 'temporary_lock', label: t('admin.securityCenter.policyActions.temporaryLock') },
  { value: 'disable_api_key', label: t('admin.securityCenter.policyActions.disableApiKey') },
  { value: 'disable_user', label: t('admin.securityCenter.policyActions.disableUser') },
  { value: 'notify_admin', label: t('admin.securityCenter.policyActions.notifyAdmin') },
  { value: 'notify_user', label: t('admin.securityCenter.policyActions.notifyUser') },
])

const lockSubjectOptions = computed(() => [
  { value: 'api_key', label: 'API Key' },
  { value: 'user', label: t('securityCenter.fields.subject') },
])

const pageElevatedRiskCount = computed(() =>
  logs.value.filter((log) => ['high', 'critical'].includes(String(log.risk_level))).length,
)

const integrityBadgeClass = computed(() => {
  const base = 'inline-flex rounded-full px-2.5 py-1 text-xs font-medium'
  if (!integrity.value) return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
  return integrity.value.valid
    ? `${base} bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
    : `${base} bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300`
})

function toDayStartIso(value: string): string {
  return new Date(`${value}T00:00:00`).toISOString()
}

function toDayEndIso(value: string): string {
  return new Date(`${value}T23:59:59.999`).toISOString()
}

function buildParams(): AdminSecurityAuditFilterParams {
  const params: AdminSecurityAuditFilterParams = {
    page: pagination.page,
    page_size: pagination.page_size,
  }
  if (filters.action) params.action = filters.action
  if (filters.result) params.result = filters.result
  if (filters.risk_level) params.risk_level = filters.risk_level
  if (filters.actor_type) params.actor_type = filters.actor_type
  if (filters.actor_id) params.actor_id = filters.actor_id
  if (filters.subject_type) params.subject_type = filters.subject_type
  if (filters.subject_id) params.subject_id = filters.subject_id
  if (filters.request_id) params.request_id = filters.request_id
  if (filters.q) params.q = filters.q
  if (filters.start_time) params.start_time = toDayStartIso(filters.start_time)
  if (filters.end_time) params.end_time = toDayEndIso(filters.end_time)
  return params
}

async function loadLogs(): Promise<void> {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await securityAPI.listAdminAuditLogs(buildParams(), { signal: controller.signal })
    if (controller.signal.aborted) return
    logs.value = res.items || []
    pagination.total = res.total || 0
    pagination.page = res.page || pagination.page
    pagination.page_size = res.page_size || pagination.page_size
    pagination.pages = res.pages || 0
  } catch (error) {
    if (controller.signal.aborted) return
    logs.value = []
    pagination.total = 0
    pagination.pages = 0
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.failedToLoad')))
  } finally {
    if (abortController === controller) loading.value = false
  }
}

async function runIntegrityCheck(): Promise<void> {
  integrityLoading.value = true
  try {
    integrity.value = await securityAPI.checkAdminIntegrity()
    appStore.showSuccess(
      integrity.value.valid
        ? t('admin.securityCenter.integrity.validToast')
        : t('admin.securityCenter.integrity.invalidToast'),
    )
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.integrity.failed')))
  } finally {
    integrityLoading.value = false
  }
}

async function loadSecurityEntities(): Promise<void> {
  entitiesLoading.value = true
  try {
    const [incidentRes, policyRes, lockRes, exportRes] = await Promise.all([
      securityAPI.listAdminIncidents({ page: 1, page_size: 5 }),
      securityAPI.listAdminPolicies({ page: 1, page_size: 20 }),
      securityAPI.listAdminLocks({ page: 1, page_size: 10, status: 'active' }),
      securityAPI.listAdminExports({ page: 1, page_size: 5 }),
    ])
    incidents.value = incidentRes.items || []
    policies.value = policyRes.items || []
    locks.value = lockRes.items || []
    exports.value = exportRes.items || []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.entitiesFailedToLoad')))
  } finally {
    entitiesLoading.value = false
  }
}

async function createExport(): Promise<void> {
  exportCreating.value = true
  try {
    const params = buildParams()
    const { page: _page, page_size: _pageSize, ...filterParams } = params
    await securityAPI.createAdminExport(filterParams)
    appStore.showSuccess(t('admin.securityCenter.exports.created'))
    await loadSecurityEntities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.exports.createFailed')))
  } finally {
    exportCreating.value = false
  }
}

async function downloadExport(item: SecurityAuditExport): Promise<void> {
  downloadingExportId.value = item.id
  try {
    const blob = await securityAPI.downloadAdminExport(item.id)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `security_audit_${item.export_key}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.exports.downloadFailed')))
  } finally {
    downloadingExportId.value = null
  }
}

function parseConditions(text: string): Record<string, unknown> | null {
  const raw = text.trim()
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      appStore.showError(t('admin.securityCenter.policies.invalidConditions'))
      return null
    }
    return parsed as Record<string, unknown>
  } catch {
    appStore.showError(t('admin.securityCenter.policies.invalidConditions'))
    return null
  }
}

async function savePolicy(): Promise<void> {
  if (!policyForm.name.trim() || !policyForm.code.trim()) {
    appStore.showError(t('admin.securityCenter.policies.nameCodeRequired'))
    return
  }
  const conditions = parseConditions(policyForm.conditionsText)
  if (!conditions) return
  const payload: SecurityPolicyRuleInput = {
    name: policyForm.name.trim(),
    code: policyForm.code.trim(),
    description: policyForm.description.trim(),
    enabled: policyForm.enabled,
    severity: policyForm.severity,
    action: policyForm.action,
    conditions,
  }
  policySaving.value = true
  try {
    if (policyForm.id) {
      await securityAPI.updateAdminPolicy(policyForm.id, payload)
      appStore.showSuccess(t('admin.securityCenter.policies.updated'))
    } else {
      await securityAPI.createAdminPolicy(payload)
      appStore.showSuccess(t('admin.securityCenter.policies.created'))
    }
    resetPolicyForm()
    await loadSecurityEntities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.policies.saveFailed')))
  } finally {
    policySaving.value = false
  }
}

function editPolicy(policy: SecurityPolicyRule): void {
  policyForm.id = policy.id
  policyForm.name = policy.name
  policyForm.code = policy.code
  policyForm.description = policy.description || ''
  policyForm.enabled = policy.enabled
  policyForm.severity = String(policy.severity || 'medium')
  policyForm.action = String(policy.action || 'observe')
  policyForm.conditionsText = JSON.stringify(policy.conditions || {}, null, 2)
}

function resetPolicyForm(): void {
  policyForm.id = null
  policyForm.name = ''
  policyForm.code = ''
  policyForm.description = ''
  policyForm.enabled = true
  policyForm.severity = 'medium'
  policyForm.action = 'observe'
  policyForm.conditionsText = '{}'
}

async function deletePolicy(policy: SecurityPolicyRule): Promise<void> {
  if (!window.confirm(t('admin.securityCenter.policies.deleteConfirm', { name: policy.name }))) return
  try {
    await securityAPI.deleteAdminPolicy(policy.id)
    appStore.showSuccess(t('admin.securityCenter.policies.deleted'))
    await loadSecurityEntities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.policies.deleteFailed')))
  }
}

async function createLock(): Promise<void> {
  if (!lockForm.subject_id || lockForm.subject_id <= 0) {
    appStore.showError(t('admin.securityCenter.locks.subjectRequired'))
    return
  }
  lockSaving.value = true
  try {
    await securityAPI.createAdminLock({
      subject_type: lockForm.subject_type,
      subject_id: lockForm.subject_id,
      reason: lockForm.reason.trim(),
    })
    appStore.showSuccess(t('admin.securityCenter.locks.created'))
    lockForm.subject_id = null
    lockForm.reason = ''
    await loadSecurityEntities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.locks.createFailed')))
  } finally {
    lockSaving.value = false
  }
}

async function unlockLock(lock: SecuritySubjectLock): Promise<void> {
  try {
    await securityAPI.unlockAdminLock(lock.id, 'manual unlock')
    appStore.showSuccess(t('admin.securityCenter.locks.unlocked'))
    await loadSecurityEntities()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.securityCenter.locks.unlockFailed')))
  }
}

function applyFilters(): void {
  pagination.page = 1
  loadLogs()
}

function resetFilters(): void {
  filters.action = ''
  filters.result = ''
  filters.risk_level = ''
  filters.actor_type = ''
  filters.actor_id = null
  filters.subject_type = ''
  filters.subject_id = null
  filters.request_id = ''
  filters.q = ''
  filters.start_time = ''
  filters.end_time = ''
  pagination.page = 1
  loadLogs()
}

function handlePageChange(page: number): void {
  pagination.page = page
  loadLogs()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  loadLogs()
}

function displayAction(log: SecurityAuditLog): string {
  return log.action || log.event_type || t('securityCenter.fields.unknownAction')
}

function formatPrincipal(type: string, id?: number | null, label?: string): string {
  const pieces = [type || t('usage.unknown')]
  if (id) pieces.push(`#${id}`)
  if (label) pieces.push(label)
  return pieces.join(' · ')
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

function exportStatusLabel(value: string): string {
  const key = `admin.securityCenter.exports.status.${value}`
  const label = t(key)
  return label === key ? value || '-' : label
}

function exportStatusBadgeClass(value: string): string {
  if (value === 'completed') return resultBadgeClass('success')
  if (value === 'failed') return resultBadgeClass('failure')
  return resultBadgeClass('denied')
}

function riskBadgeClass(value: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (value === 'critical') return `${base} bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200`
  if (value === 'high') return `${base} bg-orange-100 text-orange-800 dark:bg-orange-900/40 dark:text-orange-200`
  if (value === 'medium') return `${base} bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-200`
  return `${base} bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300`
}

onMounted(() => {
  loadLogs()
  loadSecurityEntities()
})
</script>
