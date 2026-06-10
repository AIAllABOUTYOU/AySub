<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.xaiCookieTokenImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="xai-cookie-token-import-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.xaiCookieTokenImportHint') }}
      </div>
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400">
        {{ t('admin.accounts.xaiCookieTokenImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.xaiCookieTokenImportFile') }}</label>
        <div class="flex flex-col gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ sourceSummary || t('admin.accounts.xaiCookieTokenImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">TXT / JSON / ZIP (.txt, .json, .zip)</div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openFilePicker">
              {{ t('admin.accounts.xaiCookieTokenImportChooseFiles') }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openDirectoryPicker">
              {{ t('admin.accounts.xaiCookieTokenImportChooseFolder') }}
            </button>
          </div>
        </div>
        <div v-if="fileContents.length || skippedFiles.length" class="mt-2 space-y-1 text-xs text-gray-500 dark:text-dark-400">
          <div v-if="fileContents.length">
            {{ t('admin.accounts.xaiCookieTokenImportSelectedSources', { count: fileContents.length }) }}
          </div>
          <div v-if="skippedFiles.length" class="text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.xaiCookieTokenImportSkippedSources', { count: skippedFiles.length }) }}
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept=".txt,.json,.zip,text/plain,application/json,application/zip"
          multiple
          @change="handleFileChange"
        />
        <input
          ref="directoryInput"
          type="file"
          class="hidden"
          accept=".txt,.json,.zip,text/plain,application/json,application/zip"
          webkitdirectory
          multiple
          @change="handleFileChange"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.accounts.xaiCookieTokenNamePrefix') }}</label>
          <input v-model="namePrefix" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.xaiCookieTokenBaseUrl') }}</label>
          <input v-model="baseUrl" type="text" class="input" placeholder="https://grok.com" />
        </div>
      </div>

      <div v-if="importProgress" class="space-y-2 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-900/60 dark:bg-blue-950/30">
        <div class="flex items-center justify-between text-sm text-blue-700 dark:text-blue-300">
          <span>{{ t('admin.accounts.xaiCookieTokenImportProgress', importProgress) }}</span>
          <span>{{ importProgress.percent }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-blue-100 dark:bg-blue-900/50">
          <div class="h-full rounded-full bg-blue-500 transition-all" :style="{ width: `${importProgress.percent}%` }"></div>
        </div>
      </div>

      <div v-if="result" class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.xaiCookieTokenImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.xaiCookieTokenImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800">
            <div v-for="item in errorItems" :key="item.line" class="whitespace-pre-wrap">
              {{ t('admin.accounts.xaiCookieTokenImportErrorLine', { line: item.line }) }} - {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="xai-cookie-token-import-form"
          :disabled="importing || loadingFiles"
        >
          {{ importing ? t('admin.accounts.xaiCookieTokenImporting') : t('admin.accounts.xaiCookieTokenImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { readImportTextFiles, type ImportTextFile, type ImportTextSkippedFile } from '@/utils/importTextFiles'
import type { XaiCookieTokenImportResult } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const TOKEN_IMPORT_BATCH_SIZE = 100

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const loadingFiles = ref(false)
const fileContents = ref<ImportTextFile[]>([])
const skippedFiles = ref<ImportTextSkippedFile[]>([])
const result = ref<XaiCookieTokenImportResult | null>(null)
const importProgress = ref<{
  current: number
  total: number
  processed: number
  total_items: number
  percent: number
} | null>(null)
const namePrefix = ref('')
const baseUrl = ref('https://grok.com')
const fileInput = ref<HTMLInputElement | null>(null)
const directoryInput = ref<HTMLInputElement | null>(null)

const sourceSummary = computed(() => {
  if (loadingFiles.value) return t('admin.accounts.xaiCookieTokenImportReadingFiles')
  if (!fileContents.value.length) return ''
  return t('admin.accounts.xaiCookieTokenImportSourceSummary', {
    count: fileContents.value.length,
    first: fileContents.value[0]?.name || '-'
  })
})
const errorItems = computed(() => result.value?.errors || [])

watch(
  () => props.show,
  (open) => {
    if (open) {
      fileContents.value = []
      skippedFiles.value = []
      result.value = null
      importProgress.value = null
      namePrefix.value = t('admin.accounts.xaiCookieTokenDefaultNamePrefix')
      baseUrl.value = 'https://grok.com'
      if (fileInput.value) {
        fileInput.value.value = ''
      }
      if (directoryInput.value) {
        directoryInput.value.value = ''
      }
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const openDirectoryPicker = () => {
  directoryInput.value?.click()
}

const handleFileChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = Array.from(target.files || [])
  target.value = ''
  if (!files.length) return

  loadingFiles.value = true
  result.value = null
  importProgress.value = null
  try {
    const { loaded, skipped } = await readImportTextFiles(files, {
      extensions: ['.txt', '.json'],
      messages: {
        unsupportedFile: t('admin.accounts.xaiCookieTokenImportUnsupportedFile'),
        invalidZip: t('admin.accounts.xaiCookieTokenImportInvalidZip'),
        zipReadFailed: t('admin.accounts.xaiCookieTokenImportZipReadFailed'),
        zipUnsupportedBrowser: t('admin.accounts.xaiCookieTokenImportZipUnsupportedBrowser'),
        unsupportedZipMethod: (method) => t('admin.accounts.xaiCookieTokenImportUnsupportedZipMethod', { method })
      }
    })
    fileContents.value = loaded
    skippedFiles.value = skipped
    if (loaded.length) {
      appStore.showSuccess(t('admin.accounts.xaiCookieTokenImportFilesLoaded', { count: loaded.length }))
    } else if (skipped.length) {
      appStore.showError(t('admin.accounts.xaiCookieTokenImportNoSupportedFiles'))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.xaiCookieTokenImportFailed'))
  } finally {
    loadingFiles.value = false
  }
}

const handleClose = () => {
  if (importing.value || loadingFiles.value) return
  emit('close')
}

const extractTokenFromValue = (value: unknown): string => {
  if (typeof value === 'string') return value.trim()
  if (value && typeof value === 'object') {
    const token = (value as { token?: unknown }).token
    if (typeof token === 'string') return token.trim()
  }
  return ''
}

const parseTokenFile = (text: string): string[] => {
  const normalized = text.replace(/^\ufeff/, '').trim()
  if (!normalized) return []

  if (normalized.startsWith('{') || normalized.startsWith('[')) {
    const payload = JSON.parse(normalized)
    const source = Array.isArray(payload)
      ? payload
      : Array.isArray(payload?.ssoBasic)
        ? payload.ssoBasic
        : Array.isArray(payload?.tokens)
          ? payload.tokens
          : null

    if (source) {
      return source.map(extractTokenFromValue).filter(Boolean)
    }
  }

  return normalized.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
}

const createEmptyResult = (): XaiCookieTokenImportResult => ({
  created: 0,
  skipped: 0,
  failed: 0,
  errors: []
})

const appendImportResult = (
  target: XaiCookieTokenImportResult,
  source: XaiCookieTokenImportResult,
  lineOffset: number
) => {
  target.created += source.created || 0
  target.skipped += source.skipped || 0
  target.failed += source.failed || 0
  if (source.errors?.length) {
    target.errors = [
      ...(target.errors || []),
      ...source.errors.map((item) => ({
        ...item,
        line: item.line + lineOffset
      }))
    ]
  }
}

const handleImport = async () => {
  if (!fileContents.value.length) {
    appStore.showError(t('admin.accounts.xaiCookieTokenImportSelectFile'))
    return
  }

  importing.value = true
  try {
    const tokens = fileContents.value.flatMap((item) => parseTokenFile(item.content))
    if (!tokens.length) {
      appStore.showError(t('admin.accounts.xaiCookieTokenImportEmptyFile'))
      return
    }

    const aggregate = createEmptyResult()
    const total = Math.ceil(tokens.length / TOKEN_IMPORT_BATCH_SIZE)
    result.value = aggregate
    for (let index = 0; index < total; index += 1) {
      const offset = index * TOKEN_IMPORT_BATCH_SIZE
      const batch = tokens.slice(offset, offset + TOKEN_IMPORT_BATCH_SIZE)
      importProgress.value = {
        current: index + 1,
        total,
        processed: offset,
        total_items: tokens.length,
        percent: Math.floor((offset / tokens.length) * 100)
      }
      const res = await adminAPI.accounts.importXaiCookieTokens({
        tokens: batch,
        name_prefix: namePrefix.value.trim(),
        base_url: baseUrl.value.trim(),
        name_start_index: aggregate.created
      })
      appendImportResult(aggregate, res, offset)
      result.value = { ...aggregate, errors: [...(aggregate.errors || [])] }
      importProgress.value = {
        current: index + 1,
        total,
        processed: Math.min(offset + batch.length, tokens.length),
        total_items: tokens.length,
        percent: Math.floor((Math.min(offset + batch.length, tokens.length) / tokens.length) * 100)
      }
    }
    importProgress.value = null

    const msgParams = {
      created: aggregate.created,
      skipped: aggregate.skipped,
      failed: aggregate.failed
    }
    if (aggregate.failed > 0) {
      appStore.showError(t('admin.accounts.xaiCookieTokenImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.xaiCookieTokenImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.xaiCookieTokenImportFailed'))
  } finally {
    importing.value = false
  }
}
</script>
