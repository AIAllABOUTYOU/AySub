<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.codexSessionImportTitle')"
    width="wide"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="codex-session-import-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="rounded-lg border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800 dark:border-blue-800/50 dark:bg-blue-900/20 dark:text-blue-200">
        {{ t('admin.accounts.codexSessionImportHint') }}
      </div>
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.accounts.codexSessionImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.codexSessionImportFile') }}</label>
        <div class="flex flex-col gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800 sm:flex-row sm:items-center sm:justify-between">
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ sourceSummary || t('admin.accounts.codexSessionImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">JSON / TXT / ZIP (.json, .txt, .zip)</div>
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openFilePicker">
              {{ t('admin.accounts.codexSessionImportChooseFiles') }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="loadingFiles" @click="openDirectoryPicker">
              {{ t('admin.accounts.codexSessionImportChooseFolder') }}
            </button>
          </div>
        </div>
        <div v-if="fileContents.length || skippedFiles.length" class="mt-2 space-y-1 text-xs text-gray-500 dark:text-dark-400">
          <div v-if="fileContents.length">
            {{ t('admin.accounts.codexSessionImportSelectedSources', { count: fileContents.length }) }}
          </div>
          <div v-if="skippedFiles.length" class="text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.codexSessionImportSkippedSources', { count: skippedFiles.length }) }}
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept=".json,.txt,.zip,application/json,text/plain,application/zip"
          multiple
          @change="handleFileChange"
        />
        <input
          ref="directoryInput"
          type="file"
          class="hidden"
          accept=".json,.txt,.zip,application/json,text/plain,application/zip"
          webkitdirectory
          multiple
          @change="handleFileChange"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.codexSessionImportContent') }}</label>
        <textarea
          v-model="content"
          class="input min-h-[220px] resize-y font-mono text-xs leading-5"
          spellcheck="false"
          :placeholder="t('admin.accounts.codexSessionImportPlaceholder')"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <div>
          <label class="input-label">{{ t('admin.accounts.codexSessionImportName') }}</label>
          <input v-model="name" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.concurrency') }}</label>
          <input v-model.number="concurrency" type="number" min="0" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.priority') }}</label>
          <input v-model.number="priority" type="number" min="0" class="input" />
        </div>
      </div>

      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input v-model="updateExisting" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
        <span>{{ t('admin.accounts.codexSessionImportUpdateExisting') }}</span>
      </label>

      <div v-if="result" class="space-y-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.codexSessionImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.codexSessionImportResultSummary', result) }}
        </div>
        <div v-if="detailItems.length" class="max-h-52 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800">
          <div v-for="item in detailItems" :key="`${item.kind}-${item.index}-${item.message}`" class="whitespace-pre-wrap">
            {{ item.kind }} #{{ item.index }} {{ item.name || '-' }} - {{ item.message }}
          </div>
        </div>
      </div>

      <div class="border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-400">
        <span>{{ t('admin.accounts.codexSessionImportOpenSource') }}</span>
        <a
          class="ml-1 break-all font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
          href="https://github.com/gtxx3600/GPTSession2CPAandSub2API"
          target="_blank"
          rel="noreferrer"
        >
          gtxx3600/GPTSession2CPAandSub2API
        </a>
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
          form="codex-session-import-form"
          :disabled="importing || loadingFiles"
        >
          {{ importing ? t('admin.accounts.codexSessionImporting') : t('admin.accounts.codexSessionImportButton') }}
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
import type { CodexSessionImportMessage, CodexSessionImportResult } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

type DetailItem = CodexSessionImportMessage & { kind: string }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const loadingFiles = ref(false)
const fileContents = ref<ImportTextFile[]>([])
const skippedFiles = ref<ImportTextSkippedFile[]>([])
const content = ref('')
const name = ref('')
const concurrency = ref(3)
const priority = ref(50)
const updateExisting = ref(true)
const result = ref<CodexSessionImportResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const directoryInput = ref<HTMLInputElement | null>(null)

const sourceSummary = computed(() => {
  if (loadingFiles.value) return t('admin.accounts.codexSessionImportReadingFiles')
  if (!fileContents.value.length) return ''
  return t('admin.accounts.codexSessionImportSourceSummary', {
    count: fileContents.value.length,
    first: fileContents.value[0]?.name || '-'
  })
})
const detailItems = computed<DetailItem[]>(() => [
  ...(result.value?.errors || []).map((item) => ({ ...item, kind: t('admin.accounts.codexSessionImportError') })),
  ...(result.value?.warnings || []).map((item) => ({ ...item, kind: t('admin.accounts.codexSessionImportWarningLabel') }))
])

watch(
  () => props.show,
  (open) => {
    if (open) {
      fileContents.value = []
      skippedFiles.value = []
      content.value = ''
      name.value = ''
      concurrency.value = 3
      priority.value = 50
      updateExisting.value = true
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
      extensions: ['.json', '.txt'],
      messages: {
        unsupportedFile: t('admin.accounts.codexSessionImportUnsupportedFile'),
        invalidZip: t('admin.accounts.codexSessionImportInvalidZip'),
        zipReadFailed: t('admin.accounts.codexSessionImportZipReadFailed'),
        zipUnsupportedBrowser: t('admin.accounts.codexSessionImportZipUnsupportedBrowser'),
        unsupportedZipMethod: (method) => t('admin.accounts.codexSessionImportUnsupportedZipMethod', { method })
      }
    })

    fileContents.value = loaded
    skippedFiles.value = skipped
    if (loaded.length) {
      appStore.showSuccess(t('admin.accounts.codexSessionImportFilesLoaded', { count: loaded.length }))
    } else if (skipped.length) {
      appStore.showError(t('admin.accounts.codexSessionImportNoSupportedFiles'))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.codexSessionImportReadFailed'))
  } finally {
    loadingFiles.value = false
  }
}

const handleClose = () => {
  if (importing.value || loadingFiles.value) return
  emit('close')
}

const handleImport = async () => {
  const normalizedContent = content.value.trim()
  const sourceContents = fileContents.value.map((item) => item.content.trim()).filter(Boolean)
  if (!normalizedContent && sourceContents.length === 0) {
    appStore.showError(t('admin.accounts.codexSessionImportEmpty'))
    return
  }

  importing.value = true
  try {
    const res = await adminAPI.accounts.importCodexSession({
      content: normalizedContent || undefined,
      contents: sourceContents,
      name: name.value.trim() || undefined,
      concurrency: concurrency.value,
      priority: priority.value,
      update_existing: updateExisting.value,
      skip_default_group_bind: true,
      extra: {
        import_ui: 'codex_session_compat'
      }
    })
    result.value = res

    const msgParams = {
      created: res.created,
      updated: res.updated,
      skipped: res.skipped,
      failed: res.failed
    }
    if (res.failed > 0) {
      appStore.showError(t('admin.accounts.codexSessionImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.codexSessionImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.codexSessionImportFailed'))
  } finally {
    importing.value = false
  }
}

</script>
