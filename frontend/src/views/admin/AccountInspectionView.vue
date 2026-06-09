<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
            <button type="button" class="inline-flex items-center gap-1 hover:text-gray-700 dark:hover:text-gray-200" @click="router.push('/admin/accounts')">
              <Icon name="arrowLeft" size="xs" />
              账号管理
            </button>
            <span>/</span>
            <span>账号巡检</span>
          </div>
          <h1 class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">账号巡检</h1>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="running" @click="router.push('/admin/accounts')">
            返回账号管理
          </button>
          <button type="button" class="btn btn-primary" :disabled="running" @click="runInspection">
            <Icon name="beaker" size="sm" :class="{ 'animate-pulse': running }" />
            <span>{{ running ? '巡检中' : '开始巡检' }}</span>
          </button>
        </div>
      </div>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          <label class="space-y-1.5">
            <span class="input-label">目标</span>
            <select v-model="form.target_type" class="input">
              <option value="all">全部账号</option>
              <option value="codex">Codex 账号</option>
              <option value="openai">OpenAI 账号</option>
            </select>
          </label>
          <label class="space-y-1.5">
            <span class="input-label">模型</span>
            <input v-model.trim="form.model_id" type="text" class="input" placeholder="默认测试模型" />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">并发</span>
            <input v-model.number="form.concurrency" type="number" min="1" max="20" class="input" />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">超时 ms</span>
            <input v-model.number="form.timeout_ms" type="number" min="3000" max="120000" step="1000" class="input" />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">抽样数</span>
            <input v-model.number="form.sample_size" type="number" min="0" class="input" />
          </label>
        </div>
        <div class="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-5">
          <label class="space-y-1.5">
            <span class="input-label">额度阈值 %</span>
            <input v-model.number="form.used_percent_threshold" type="number" min="1" max="1000" step="1" class="input" />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">平台</span>
            <select v-model="filters.platform" class="input">
              <option value="">自动</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
              <option value="gemini">Gemini</option>
              <option value="antigravity">Antigravity</option>
              <option value="xai">xAI</option>
            </select>
          </label>
          <label class="space-y-1.5">
            <span class="input-label">类型</span>
            <select v-model="filters.type" class="input">
              <option value="">自动</option>
              <option value="oauth">OAuth</option>
              <option value="apikey">API Key</option>
              <option value="setup-token">Setup Token</option>
              <option value="cookie">Cookie</option>
              <option value="upstream">Upstream</option>
            </select>
          </label>
          <label class="space-y-1.5">
            <span class="input-label">状态</span>
            <select v-model="filters.status" class="input">
              <option value="">全部</option>
              <option value="active">active</option>
              <option value="disabled">disabled</option>
              <option value="error">error</option>
            </select>
          </label>
          <label class="space-y-1.5">
            <span class="input-label">分组 ID</span>
            <input v-model.trim="filters.group" type="text" class="input" placeholder="留空或 ungrouped" />
          </label>
          <label class="space-y-1.5">
            <span class="input-label">搜索</span>
            <input v-model.trim="filters.search" type="text" class="input" placeholder="账号名 / 备注" />
          </label>
        </div>
      </section>

      <section class="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <div v-for="stat in statCards" :key="stat.label" class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ stat.label }}</div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ stat.value }}</div>
          <div class="mt-1 text-xs" :class="stat.class">{{ stat.sub }}</div>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-3 border-b border-gray-200 p-4 dark:border-dark-700 lg:flex-row lg:items-center lg:justify-between">
          <div class="flex flex-wrap gap-2">
            <button
              v-for="tab in tabs"
              :key="tab.key"
              type="button"
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              :class="activeTab === tab.key ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600'"
              @click="activeTab = tab.key"
            >
              {{ tab.label }} {{ tab.count }}
            </button>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <button type="button" class="btn btn-secondary btn-sm" :disabled="actionLoading || idsForAction('enable').length === 0" @click="applyAction('enable')">
              启用建议
            </button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="actionLoading || idsForAction('disable').length === 0" @click="applyAction('disable')">
              停用建议
            </button>
            <button type="button" class="btn btn-danger btn-sm" :disabled="actionLoading || idsForAction('delete').length === 0" @click="applyAction('delete')">
              删除建议
            </button>
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/50">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">账号</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">当前状态</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">巡检</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">额度</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">耗时</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">建议</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">原因</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-if="filteredItems.length === 0">
                <td colspan="7" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ running ? '正在生成结果' : '暂无巡检结果' }}
                </td>
              </tr>
              <tr v-for="item in filteredItems" :key="item.account_id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                <td class="px-4 py-3">
                  <div class="max-w-[280px] truncate text-sm font-medium text-gray-900 dark:text-white">{{ item.account_name || `#${item.account_id}` }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">#{{ item.account_id }} · {{ item.platform }} / {{ item.type }}</div>
                </td>
                <td class="px-4 py-3">
                  <div class="flex flex-wrap gap-1.5">
                    <span class="rounded-md px-2 py-1 text-xs font-medium" :class="statusBadgeClass(item.current_status)">{{ item.current_status }}</span>
                    <span class="rounded-md px-2 py-1 text-xs font-medium" :class="item.runtime_available ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">
                      {{ item.runtime_available ? '可调度' : '不可调度' }}
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3">
                  <span class="rounded-md px-2 py-1 text-xs font-medium" :class="inspectionBadgeClass(item.status)">
                    {{ inspectionLabel(item.status) }}
                  </span>
                  <span v-if="item.http_status" class="ml-2 text-xs text-gray-500 dark:text-gray-400">HTTP {{ item.http_status }}</span>
                </td>
                <td class="px-4 py-3">
                  <div class="text-sm text-gray-700 dark:text-gray-300">
                    {{ quotaLabel(item) }}
                  </div>
                  <div v-if="item.five_hour_used_percent !== undefined || item.usage_probe_status" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    <span v-if="item.five_hour_used_percent !== undefined">5h {{ formatPercent(item.five_hour_used_percent) }}</span>
                    <span v-if="item.five_hour_used_percent !== undefined && item.usage_probe_status"> · </span>
                    <span v-if="item.usage_probe_status">{{ usageProbeLabel(item.usage_probe_status) }}</span>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.latency_ms ? `${item.latency_ms} ms` : '-' }}</td>
                <td class="px-4 py-3">
                  <span class="rounded-md px-2 py-1 text-xs font-medium" :class="actionBadgeClass(item.suggested_action)">
                    {{ actionLabel(item.suggested_action) }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <div class="max-w-[420px] text-sm text-gray-700 dark:text-gray-300">
                    {{ item.suggested_reason || '-' }}
                  </div>
                  <div v-if="item.error_message" class="mt-1 max-w-[420px] truncate text-xs text-red-600 dark:text-red-300">
                    {{ item.error_message }}
                  </div>
                  <div v-if="item.usage_probe_error" class="mt-1 max-w-[420px] truncate text-xs text-amber-600 dark:text-amber-300">
                    {{ item.usage_probe_error }}
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">巡检日志</h2>
          <button type="button" class="text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200" @click="logs = []">清空</button>
        </div>
        <div class="max-h-48 overflow-auto rounded-lg bg-gray-950 p-3 font-mono text-xs leading-5 text-gray-100">
          <div v-if="logs.length === 0" class="text-gray-500">No logs</div>
          <div v-for="(line, index) in logs" :key="index">{{ line }}</div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type {
  AccountInspectionAction,
  AccountInspectionItem,
  AccountInspectionRunResult
} from '@/api/admin/accounts'
import { useAppStore } from '@/stores/app'

type TabKey = 'all' | 'failed' | 'delete' | 'disable' | 'enable' | 'reauth' | 'success'

const router = useRouter()
const appStore = useAppStore()

const running = ref(false)
const actionLoading = ref(false)
const result = ref<AccountInspectionRunResult | null>(null)
const activeTab = ref<TabKey>('all')
const logs = ref<string[]>([])

const form = reactive({
  target_type: 'all' as 'codex' | 'openai' | 'all',
  model_id: '',
  concurrency: 4,
  timeout_ms: 15000,
  sample_size: 0,
  used_percent_threshold: 100
})

const filters = reactive({
  platform: '',
  type: '',
  status: '',
  group: '',
  search: '',
  privacy_mode: ''
})

const items = computed(() => result.value?.items ?? [])
const summary = computed(() => result.value?.summary)

const statCards = computed(() => [
  { label: '账号数', value: summary.value?.total_accounts ?? 0, sub: `已测 ${summary.value?.tested ?? 0}`, class: 'text-gray-500 dark:text-gray-400' },
  { label: '成功', value: summary.value?.success ?? 0, sub: '可继续使用', class: 'text-emerald-600 dark:text-emerald-300' },
  { label: '失败', value: summary.value?.failed ?? 0, sub: `跳过 ${summary.value?.skipped ?? 0}`, class: 'text-red-600 dark:text-red-300' },
  { label: '需重认', value: summary.value?.suggest_reauth ?? 0, sub: '认证文件失效', class: 'text-amber-600 dark:text-amber-300' },
  { label: '建议处理', value: (summary.value?.suggest_delete ?? 0) + (summary.value?.suggest_disable ?? 0) + (summary.value?.suggest_enable ?? 0), sub: `删 ${summary.value?.suggest_delete ?? 0} / 停 ${summary.value?.suggest_disable ?? 0} / 启 ${summary.value?.suggest_enable ?? 0}`, class: 'text-primary-600 dark:text-primary-300' }
])

const tabs = computed(() => [
  { key: 'all' as const, label: '全部', count: items.value.length },
  { key: 'failed' as const, label: '失败', count: items.value.filter(i => i.status === 'failed').length },
  { key: 'delete' as const, label: '建议删除', count: idsForAction('delete').length },
  { key: 'disable' as const, label: '建议停用', count: idsForAction('disable').length },
  { key: 'enable' as const, label: '建议启用', count: idsForAction('enable').length },
  { key: 'reauth' as const, label: '需重认', count: idsForAction('reauth').length },
  { key: 'success' as const, label: '成功', count: items.value.filter(i => i.status === 'success').length }
])

const filteredItems = computed(() => {
  switch (activeTab.value) {
    case 'failed':
      return items.value.filter(i => i.status === 'failed')
    case 'success':
      return items.value.filter(i => i.status === 'success')
    case 'delete':
    case 'disable':
    case 'enable':
    case 'reauth':
      return items.value.filter(i => i.suggested_action === activeTab.value)
    default:
      return items.value
  }
})

function compactFilters() {
  return Object.fromEntries(
    Object.entries(filters).filter(([, value]) => String(value ?? '').trim() !== '')
  )
}

function logLine(message: string) {
  const time = new Date().toLocaleTimeString()
  logs.value.push(`[${time}] ${message}`)
}

async function runInspection() {
  running.value = true
  activeTab.value = 'all'
  result.value = null
  logLine(`start target=${form.target_type} concurrency=${form.concurrency} timeout=${form.timeout_ms}`)
  try {
    const payload = {
      target_type: form.target_type,
      model_id: form.model_id || undefined,
      concurrency: Number(form.concurrency) || 4,
      timeout_ms: Number(form.timeout_ms) || 15000,
      sample_size: Number(form.sample_size) || 0,
      used_percent_threshold: Number(form.used_percent_threshold) || 100,
      filters: compactFilters()
    }
    result.value = await adminAPI.accounts.runInspection(payload)
    logLine(`done total=${result.value.summary.total_accounts} success=${result.value.summary.success} failed=${result.value.summary.failed}`)
    appStore.showSuccess('巡检完成')
  } catch (error) {
    console.error('Failed to run account inspection:', error)
    logLine(`error ${error instanceof Error ? error.message : String(error)}`)
    appStore.showError('巡检失败')
  } finally {
    running.value = false
  }
}

function idsForAction(action: AccountInspectionAction) {
  return items.value.filter(i => i.suggested_action === action).map(i => i.account_id)
}

async function applyAction(action: 'enable' | 'disable' | 'delete') {
  const ids = idsForAction(action)
  if (ids.length === 0) return
  const label = actionLabel(action)
  if (!window.confirm(`确认${label} ${ids.length} 个账号？`)) return

  actionLoading.value = true
  logLine(`apply ${action} ids=${ids.length}`)
  try {
    if (action === 'delete') {
      await adminAPI.accounts.batchDelete(ids)
    } else {
      await adminAPI.accounts.bulkUpdate(ids, { schedulable: action === 'enable' })
    }
    appStore.showSuccess(`${label}完成`)
    logLine(`apply ${action} done`)
    await runInspection()
  } catch (error) {
    console.error('Failed to apply inspection action:', error)
    logLine(`apply ${action} error ${error instanceof Error ? error.message : String(error)}`)
    appStore.showError(`${label}失败`)
  } finally {
    actionLoading.value = false
  }
}

function inspectionLabel(status: AccountInspectionItem['status']) {
  if (status === 'success') return '成功'
  if (status === 'failed') return '失败'
  if (status === 'skipped') return '跳过'
  return '等待'
}

function actionLabel(action: AccountInspectionAction) {
  if (action === 'enable') return '启用'
  if (action === 'disable') return '停用'
  if (action === 'delete') return '删除'
  if (action === 'reauth') return '重认'
  return '保留'
}

function formatPercent(value?: number) {
  if (value === undefined || Number.isNaN(value)) return '-'
  return `${value.toFixed(value >= 10 ? 1 : 2)}%`
}

function quotaLabel(item: AccountInspectionItem) {
  if (item.used_percent === undefined) {
    return item.usage_probe_status ? '未返回长期额度' : '-'
  }
  return `${item.quota_window || '长期额度'} ${formatPercent(item.used_percent)}`
}

function usageProbeLabel(status: NonNullable<AccountInspectionItem['usage_probe_status']>) {
  if (status === 'success') return '额度已探测'
  return '额度探测失败'
}

function statusBadgeClass(status: string) {
  if (status === 'active') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (status === 'error') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

function inspectionBadgeClass(status: AccountInspectionItem['status']) {
  if (status === 'success') return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
  if (status === 'failed') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  if (status === 'skipped') return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  return 'bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
}

function actionBadgeClass(action: AccountInspectionAction) {
  if (action === 'delete') return 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300'
  if (action === 'disable') return 'bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
  if (action === 'enable') return 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
  if (action === 'reauth') return 'bg-purple-50 text-purple-700 dark:bg-purple-900/20 dark:text-purple-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}
</script>
