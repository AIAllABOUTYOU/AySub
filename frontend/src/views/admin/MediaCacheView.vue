<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 p-5 dark:border-dark-700 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div class="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="database" size="lg" />
              </div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
                {{ t('admin.mediaCache.title') }}
              </h1>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                {{ t('admin.mediaCache.description') }}
              </p>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:min-w-64">
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('common.total') }}</div>
                <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ pagination.total.toLocaleString() }}</div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.pageSizeTotal') }}</div>
                <div class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">{{ pageBytes }}</div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="card space-y-4 p-4 sm:p-5">
        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6">
          <div>
            <label class="input-label">{{ t('admin.mediaCache.filters.type') }}</label>
            <Select v-model="filters.type" :options="typeOptions" @change="applyFilters" />
          </div>
          <div class="lg:col-span-2">
            <label class="input-label">{{ t('common.search') }}</label>
            <input v-model.trim="filters.search" class="input" :placeholder="t('admin.mediaCache.filters.searchPlaceholder')" @keyup.enter="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.mediaCache.filters.olderThan') }}</label>
            <Select v-model="filters.older_than" :options="olderThanOptions" @change="applyFilters" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.mediaCache.cleanup.limit') }}</label>
            <input v-model.number="cleanupLimit" class="input" type="number" min="0" step="1" />
          </div>
          <div class="flex items-end gap-2">
            <button class="btn btn-secondary flex-1" :disabled="loading" @click="loadItems">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('common.refresh') }}
            </button>
            <button class="btn btn-secondary flex-1" @click="resetFilters">{{ t('common.reset') }}</button>
          </div>
        </div>

        <div class="flex flex-col gap-2 sm:flex-row sm:justify-end">
          <button class="btn btn-danger" :disabled="cleanupLoading" @click="cleanupByFilter">
            {{ cleanupLoading ? t('common.processing') : t('admin.mediaCache.cleanup.filtered') }}
          </button>
          <button class="btn btn-secondary" :disabled="orphanCleanupLoading" @click="cleanupOrphans">
            {{ orphanCleanupLoading ? t('common.processing') : t('admin.mediaCache.cleanup.orphans') }}
          </button>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div v-if="loading" class="flex justify-center py-12">
          <LoadingSpinner />
        </div>

        <div v-else-if="items.length === 0" class="p-8 text-center">
          <div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <Icon name="database" size="lg" />
          </div>
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.mediaCache.empty.title') }}</h2>
          <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.empty.description') }}</p>
        </div>

        <template v-else>
          <div class="hidden overflow-x-auto lg:block">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-800">
                <tr>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.columns.preview') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.columns.file') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.columns.type') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.columns.size') }}</th>
                  <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('admin.mediaCache.columns.modifiedAt') }}</th>
                  <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">{{ t('common.actions') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="item in items" :key="`${item.type}:${item.id}`" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                  <td class="px-4 py-3">
                    <a :href="item.url" target="_blank" rel="noopener noreferrer" class="block h-14 w-20 overflow-hidden rounded border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-800">
                      <img v-if="item.type === 'image'" :src="item.url" :alt="item.file_name" class="h-full w-full object-cover" loading="lazy" />
                      <div v-else class="flex h-full w-full items-center justify-center text-gray-500 dark:text-gray-400">
                        <Icon name="play" size="md" />
                      </div>
                    </a>
                  </td>
                  <td class="max-w-md px-4 py-3">
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.file_name }}</div>
                    <div class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ item.id }}</div>
                    <div class="mt-1 truncate text-xs text-gray-400 dark:text-gray-500">{{ item.path }}</div>
                  </td>
                  <td class="px-4 py-3">
                    <span :class="typeBadgeClass(item.type)">{{ typeLabel(item.type) }}</span>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.content_type || '-' }}</div>
                  </td>
                  <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatBytes(item.size, 1) }}</td>
                  <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatUnixTime(item.modified_at) }}</td>
                  <td class="px-4 py-3 text-right">
                    <button class="btn btn-danger btn-sm" :disabled="deletingKey === `${item.type}:${item.id}`" @click="deleteItem(item)">
                      {{ deletingKey === `${item.type}:${item.id}` ? t('common.processing') : t('common.delete') }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="grid gap-3 p-4 lg:hidden">
            <article v-for="item in items" :key="`${item.type}:${item.id}`" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="flex gap-3">
                <a :href="item.url" target="_blank" rel="noopener noreferrer" class="h-16 w-20 flex-shrink-0 overflow-hidden rounded border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-800">
                  <img v-if="item.type === 'image'" :src="item.url" :alt="item.file_name" class="h-full w-full object-cover" loading="lazy" />
                  <div v-else class="flex h-full w-full items-center justify-center text-gray-500 dark:text-gray-400">
                    <Icon name="play" size="md" />
                  </div>
                </a>
                <div class="min-w-0 flex-1">
                  <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.file_name }}</div>
                  <div class="mt-1 flex flex-wrap gap-2">
                    <span :class="typeBadgeClass(item.type)">{{ typeLabel(item.type) }}</span>
                    <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatBytes(item.size, 1) }}</span>
                  </div>
                  <div class="mt-2 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ item.id }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatUnixTime(item.modified_at) }}</div>
                </div>
              </div>
              <div class="mt-3 flex justify-end">
                <button class="btn btn-danger btn-sm" :disabled="deletingKey === `${item.type}:${item.id}`" @click="deleteItem(item)">
                  {{ deletingKey === `${item.type}:${item.id}` ? t('common.processing') : t('common.delete') }}
                </button>
              </div>
            </article>
          </div>
        </template>
      </section>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { MediaCacheItem, MediaCacheType } from '@/api/admin/mediaCache'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatBytes, formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<MediaCacheItem[]>([])
const loading = ref(false)
const cleanupLoading = ref(false)
const orphanCleanupLoading = ref(false)
const deletingKey = ref('')
let abortController: AbortController | null = null

const filters = reactive<{
  type: MediaCacheType
  search: string
  older_than: string
}>({
  type: 'all',
  search: '',
  older_than: '',
})

const cleanupLimit = ref(100)

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})

const typeOptions = computed(() => [
  { value: 'all', label: t('admin.mediaCache.types.all') },
  { value: 'image', label: t('admin.mediaCache.types.image') },
  { value: 'video', label: t('admin.mediaCache.types.video') },
])

const olderThanOptions = computed(() => [
  { value: '', label: t('admin.mediaCache.olderThan.none') },
  { value: '24h', label: t('admin.mediaCache.olderThan.day') },
  { value: '168h', label: t('admin.mediaCache.olderThan.week') },
  { value: '720h', label: t('admin.mediaCache.olderThan.month') },
])

const pageBytes = computed(() => formatBytes(items.value.reduce((sum, item) => sum + (item.size || 0), 0), 1))

async function loadItems(): Promise<void> {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await adminAPI.mediaCache.list({
      page: pagination.page,
      page_size: pagination.page_size,
      type: filters.type,
      search: filters.search || undefined,
      older_than: filters.older_than || undefined,
    }, { signal: controller.signal })
    if (controller.signal.aborted) return
    items.value = res.items || []
    pagination.total = res.total || 0
    pagination.page = res.page || pagination.page
    pagination.page_size = res.page_size || pagination.page_size
  } catch (error) {
    if (controller.signal.aborted) return
    items.value = []
    pagination.total = 0
    appStore.showError(extractApiErrorMessage(error, t('admin.mediaCache.loadFailed')))
  } finally {
    if (abortController === controller) loading.value = false
  }
}

function applyFilters(): void {
  pagination.page = 1
  loadItems()
}

function resetFilters(): void {
  filters.type = 'all'
  filters.search = ''
  filters.older_than = ''
  pagination.page = 1
  loadItems()
}

async function deleteItem(item: MediaCacheItem): Promise<void> {
  if (!window.confirm(t('admin.mediaCache.deleteConfirm', { file: item.file_name }))) return
  deletingKey.value = `${item.type}:${item.id}`
  try {
    await adminAPI.mediaCache.deleteItem(item.type, item.id)
    appStore.showSuccess(t('admin.mediaCache.deleted'))
    await loadItems()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.mediaCache.deleteFailed')))
  } finally {
    deletingKey.value = ''
  }
}

async function cleanupByFilter(): Promise<void> {
  if (!filters.older_than) {
    appStore.showError(t('admin.mediaCache.cleanup.requireOlderThan'))
    return
  }
  if (!window.confirm(t('admin.mediaCache.cleanup.filteredConfirm'))) return
  cleanupLoading.value = true
  try {
    const result = await adminAPI.mediaCache.cleanup({
      type: filters.type,
      older_than: filters.older_than,
      limit: cleanupLimit.value > 0 ? cleanupLimit.value : 0,
    })
    appStore.showSuccess(t('admin.mediaCache.cleanup.result', { deleted: result.deleted, skipped: result.skipped }))
    await loadItems()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.mediaCache.cleanup.failed')))
  } finally {
    cleanupLoading.value = false
  }
}

async function cleanupOrphans(): Promise<void> {
  if (!window.confirm(t('admin.mediaCache.cleanup.orphansConfirm'))) return
  orphanCleanupLoading.value = true
  try {
    const result = await adminAPI.mediaCache.cleanupOrphans()
    appStore.showSuccess(t('admin.mediaCache.cleanup.result', { deleted: result.deleted, skipped: result.skipped }))
    await loadItems()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.mediaCache.cleanup.failed')))
  } finally {
    orphanCleanupLoading.value = false
  }
}

function handlePageChange(page: number): void {
  pagination.page = page
  loadItems()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  loadItems()
}

function formatUnixTime(value: number): string {
  if (!value) return '-'
  return formatDateTime(new Date(value * 1000))
}

function typeLabel(value: string): string {
  const key = `admin.mediaCache.types.${value}`
  const label = t(key)
  return label === key ? value : label
}

function typeBadgeClass(value: string): string {
  const base = 'inline-flex rounded-full px-2 py-0.5 text-xs font-medium'
  if (value === 'video') return `${base} bg-purple-50 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300`
  if (value === 'image') return `${base} bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  return `${base} bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300`
}

onMounted(() => {
  loadItems()
})
</script>
