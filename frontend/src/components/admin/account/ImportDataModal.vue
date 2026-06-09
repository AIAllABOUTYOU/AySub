<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex flex-col gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800 sm:flex-row sm:items-center sm:justify-between"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ sourceSummary || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">JSON / ZIP (.json, .zip)</div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openFilePicker">
              {{ t('admin.accounts.dataImportChooseFiles') }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openDirectoryPicker">
              {{ t('admin.accounts.dataImportChooseFolder') }}
            </button>
          </div>
        </div>
        <div v-if="fileContents.length || skippedFiles.length" class="mt-2 space-y-1 text-xs text-gray-500 dark:text-dark-400">
          <div v-if="fileContents.length">
            {{ t('admin.accounts.dataImportSelectedSources', { count: fileContents.length }) }}
          </div>
          <div v-if="skippedFiles.length" class="text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.dataImportSkippedSources', { count: skippedFiles.length }) }}
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json,.zip,application/zip"
          multiple
          @change="handleFileChange"
        />
        <input
          ref="directoryInput"
          type="file"
          class="hidden"
          accept="application/json,.json,.zip,application/zip"
          webkitdirectory
          multiple
          @change="handleFileChange"
        />
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
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
          form="import-data-form"
          :disabled="importing || loadingFiles"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
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
import type { AdminDataImportResult, AdminDataPayload } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const loadingFiles = ref(false)
const fileContents = ref<ImportTextFile[]>([])
const skippedFiles = ref<ImportTextSkippedFile[]>([])
const result = ref<AdminDataImportResult | null>(null)

const fileInput = ref<HTMLInputElement | null>(null)
const directoryInput = ref<HTMLInputElement | null>(null)
const sourceSummary = computed(() => {
  if (loadingFiles.value) return t('admin.accounts.dataImportReadingFiles')
  if (!fileContents.value.length) return ''
  return t('admin.accounts.dataImportSourceSummary', {
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
  try {
    const { loaded, skipped } = await readImportTextFiles(files, {
      extensions: ['.json'],
      messages: {
        unsupportedFile: t('admin.accounts.dataImportUnsupportedFile'),
        invalidZip: t('admin.accounts.dataImportInvalidZip'),
        zipReadFailed: t('admin.accounts.dataImportZipReadFailed'),
        zipUnsupportedBrowser: t('admin.accounts.dataImportZipUnsupportedBrowser'),
        unsupportedZipMethod: (method) => t('admin.accounts.dataImportUnsupportedZipMethod', { method })
      }
    })
    fileContents.value = loaded
    skippedFiles.value = skipped
    if (loaded.length) {
      appStore.showSuccess(t('admin.accounts.dataImportFilesLoaded', { count: loaded.length }))
    } else if (skipped.length) {
      appStore.showError(t('admin.accounts.dataImportNoSupportedFiles'))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
  } finally {
    loadingFiles.value = false
  }
}

const handleClose = () => {
  if (importing.value || loadingFiles.value) return
  emit('close')
}

const handleImport = async () => {
  if (!fileContents.value.length) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  try {
    const dataPayload = mergeAdminDataPayloads(fileContents.value.map((item) => parseAdminDataPayload(item.content)))

    const res = await adminAPI.accounts.importData({
      data: dataPayload,
      skip_default_group_bind: true
    })

    result.value = res

    const msgParams: Record<string, unknown> = {
      account_created: res.account_created,
      account_failed: res.account_failed,
      proxy_created: res.proxy_created,
      proxy_reused: res.proxy_reused,
      proxy_failed: res.proxy_failed,
    }
    if (res.account_failed > 0 || res.proxy_failed > 0) {
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      appStore.showError(t('admin.accounts.dataImportParseFailed'))
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
    }
  } finally {
    importing.value = false
  }
}

const parseAdminDataPayload = (text: string): AdminDataPayload => {
  const parsed = JSON.parse(text)
  const payload = parsed?.data && Array.isArray(parsed.data.accounts) ? parsed.data : parsed
  if (!payload || !Array.isArray(payload.accounts) || !Array.isArray(payload.proxies)) {
    throw new Error(t('admin.accounts.dataImportParseFailed'))
  }
  return payload as AdminDataPayload
}

const mergeAdminDataPayloads = (payloads: AdminDataPayload[]): AdminDataPayload => {
  return {
    type: payloads.find((payload) => payload.type)?.type,
    version: payloads.find((payload) => typeof payload.version === 'number')?.version,
    exported_at: payloads.find((payload) => payload.exported_at)?.exported_at || new Date().toISOString(),
    proxies: payloads.flatMap((payload) => payload.proxies || []),
    accounts: payloads.flatMap((payload) => payload.accounts || [])
  }
}
</script>
