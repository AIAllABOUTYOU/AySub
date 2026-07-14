<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.codexSessionImportTitle')"
    width="wide"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="codex-session-import-form" @submit.prevent="handleImport">
      <fieldset class="min-w-0 space-y-4" :disabled="importing || loadingFiles">
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
            <button type="button" class="btn btn-secondary" :disabled="importing || loadingFiles" @click="openFilePicker">
              {{ t('admin.accounts.codexSessionImportChooseFiles') }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="importing || loadingFiles" @click="openDirectoryPicker">
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
          @input="clearImportPreview"
        />
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <div>
          <label class="input-label">{{ t('admin.accounts.codexSessionImportName') }}</label>
          <input v-model="name" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.codexSessionImportAccountConcurrency') }}</label>
          <input
            v-model.number="concurrency"
            data-testid="codex-import-concurrency"
            type="number"
            min="0"
            step="1"
            class="input"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.priority') }}</label>
          <input
            v-model.number="priority"
            data-testid="codex-import-priority"
            type="number"
            min="0"
            step="1"
            class="input"
          />
        </div>
      </div>

      <GroupSelector v-model="groupIds" :groups="groups" platform="openai" />

      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input v-model="updateExisting" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
        <span>{{ t('admin.accounts.codexSessionImportUpdateExisting') }}</span>
      </label>

      <div
        v-if="previewResult?.http_status_401.length"
        class="space-y-3 rounded-xl border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950/30"
        data-testid="codex-import-http-401-review"
      >
        <div>
          <div class="font-medium text-amber-900 dark:text-amber-200">
            {{ t('admin.accounts.codexSessionImportHTTP401Title', { count: previewResult.http_status_401.length }) }}
          </div>
          <p class="mt-1 text-sm text-amber-800 dark:text-amber-300">
            {{ t('admin.accounts.codexSessionImportHTTP401Description') }}
          </p>
        </div>
        <div class="max-h-48 overflow-auto rounded-lg border border-amber-200 bg-white/70 p-3 font-mono text-xs dark:border-amber-900 dark:bg-dark-800/70">
          <div
            v-for="item in previewResult.http_status_401"
            :key="item.index"
            class="whitespace-pre-wrap text-gray-700 dark:text-dark-200"
          >
            #{{ item.index }} {{ item.name || item.email || item.account_id || '-' }} - HTTP {{ item.http_status }}
          </div>
        </div>
        <div class="flex flex-wrap justify-end gap-2">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="importing"
            data-testid="codex-import-include-401"
            @click="handleHTTP401Decision(false)"
          >
            {{ t('admin.accounts.codexSessionImportHTTP401Include') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="importing"
            data-testid="codex-import-filter-401"
            @click="handleHTTP401Decision(true)"
          >
            {{ t('admin.accounts.codexSessionImportHTTP401Filter') }}
          </button>
        </div>
      </div>

      <div v-if="importProgress" class="space-y-2 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-900/60 dark:bg-blue-950/30">
        <div class="flex items-center justify-between text-sm text-blue-700 dark:text-blue-300">
          <span>
            {{
              t(
                importProgress.phase === 'preview'
                  ? 'admin.accounts.codexSessionImportPreviewProgress'
                  : 'admin.accounts.codexSessionImportProgress',
                importProgress
              )
            }}
          </span>
          <span>{{ importProgress.percent }}%</span>
        </div>
        <div class="h-2 overflow-hidden rounded-full bg-blue-100 dark:bg-blue-900/50">
          <div class="h-full rounded-full bg-blue-500 transition-all" :style="{ width: `${importProgress.percent}%` }"></div>
        </div>
      </div>

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
      </fieldset>
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
import GroupSelector from '@/components/common/GroupSelector.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { readImportTextFiles, type ImportTextFile, type ImportTextSkippedFile } from '@/utils/importTextFiles'
import type {
  AdminGroup,
  CodexSessionImportMessage,
  CodexSessionImportPreviewResult,
  CodexSessionImportRequest,
  CodexSessionImportResult
} from '@/types'

interface Props {
  show: boolean
  groups: AdminGroup[]
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

type DetailItem = CodexSessionImportMessage & { kind: string }

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const CODEX_IMPORT_BATCH_SIZE = 50

const { t } = useI18n()
const appStore = useAppStore()

const importing = ref(false)
const loadingFiles = ref(false)
const fileContents = ref<ImportTextFile[]>([])
const skippedFiles = ref<ImportTextSkippedFile[]>([])
const content = ref('')
const name = ref('')
const concurrency = ref<number | ''>(3)
const priority = ref<number | ''>(50)
const groupIds = ref<number[]>([])
const updateExisting = ref(true)
const result = ref<CodexSessionImportResult | null>(null)
const previewResult = ref<CodexSessionImportPreviewResult | null>(null)
const importProgress = ref<{
  phase: 'preview' | 'import'
  current: number
  total: number
  processed: number
  total_items: number
  percent: number
} | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const directoryInput = ref<HTMLInputElement | null>(null)
let sourceGeneration = 0
let fileReadGeneration = 0
const stalePreview = Symbol('stale-preview')

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
      groupIds.value = []
      updateExisting.value = true
      result.value = null
      previewResult.value = null
      importProgress.value = null
      sourceGeneration += 1
      fileReadGeneration += 1
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

  const generation = ++fileReadGeneration
  sourceGeneration += 1

  loadingFiles.value = true
  result.value = null
  previewResult.value = null
  importProgress.value = null
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

    if (generation !== fileReadGeneration) return
    fileContents.value = loaded
    skippedFiles.value = skipped
    if (loaded.length) {
      appStore.showSuccess(t('admin.accounts.codexSessionImportFilesLoaded', { count: loaded.length }))
    } else if (skipped.length) {
      appStore.showError(t('admin.accounts.codexSessionImportNoSupportedFiles'))
    }
  } catch (error: any) {
    if (generation !== fileReadGeneration) return
    appStore.showError(error?.message || t('admin.accounts.codexSessionImportReadFailed'))
  } finally {
    if (generation === fileReadGeneration) {
      loadingFiles.value = false
    }
  }
}

const handleClose = () => {
  if (importing.value || loadingFiles.value) return
  emit('close')
}

const clearImportPreview = () => {
  sourceGeneration += 1
  previewResult.value = null
}

type CodexImportSource = {
  content: string
  kind: 'pasted' | 'file'
}

type CodexImportBatch = {
  sources: CodexImportSource[]
  entryStart?: number
  entryLimit?: number
  itemCount: number
}

type CodexImportOptions = {
  sources: CodexImportSource[]
  concurrency?: number
  priority?: number
  name?: string
  groupIds: number[]
  updateExisting: boolean
}

const createEmptyResult = (): CodexSessionImportResult => ({
  total: 0,
  created: 0,
  updated: 0,
  skipped: 0,
  failed: 0,
  items: [],
  warnings: [],
  errors: []
})

const appendImportResult = (target: CodexSessionImportResult, source: CodexSessionImportResult) => {
  target.total += source.total || 0
  target.created += source.created || 0
  target.updated += source.updated || 0
  target.skipped += source.skipped || 0
  target.failed += source.failed || 0
  target.items = [...(target.items || []), ...(source.items || [])]
  target.warnings = [...(target.warnings || []), ...(source.warnings || [])]
  target.errors = [...(target.errors || []), ...(source.errors || [])]
}

const cloneImportResult = (source: CodexSessionImportResult): CodexSessionImportResult => ({
  ...source,
  items: [...(source.items || [])],
  warnings: [...(source.warnings || [])],
  errors: [...(source.errors || [])]
})

const chunkSources = (sources: CodexImportSource[]): CodexImportSource[][] => {
  const batches: CodexImportSource[][] = []
  for (let index = 0; index < sources.length; index += CODEX_IMPORT_BATCH_SIZE) {
    batches.push(sources.slice(index, index + CODEX_IMPORT_BATCH_SIZE))
  }
  return batches
}

const buildImportBatches = (sources: CodexImportSource[], sourceEntryCounts: number[]): CodexImportBatch[] => {
  if (sources.length !== sourceEntryCounts.length) {
    throw new Error(t('admin.accounts.codexSessionImportPreviewMismatch'))
  }

  const batches: CodexImportBatch[] = []
  let pendingSources: CodexImportSource[] = []
  let pendingItemCount = 0

  const flushPending = () => {
    if (pendingItemCount === 0) return
    batches.push({ sources: pendingSources, itemCount: pendingItemCount })
    pendingSources = []
    pendingItemCount = 0
  }

  sources.forEach((source, sourceIndex) => {
    const entryCount = sourceEntryCounts[sourceIndex]
    if (!Number.isInteger(entryCount) || entryCount < 0) {
      throw new Error(t('admin.accounts.codexSessionImportPreviewMismatch'))
    }
    if (entryCount === 0) return

    if (entryCount > CODEX_IMPORT_BATCH_SIZE) {
      flushPending()
      for (let entryStart = 0; entryStart < entryCount; entryStart += CODEX_IMPORT_BATCH_SIZE) {
        const itemCount = Math.min(CODEX_IMPORT_BATCH_SIZE, entryCount - entryStart)
        batches.push({
          sources: [source],
          entryStart,
          entryLimit: itemCount,
          itemCount
        })
      }
      return
    }

    if (pendingItemCount + entryCount > CODEX_IMPORT_BATCH_SIZE) {
      flushPending()
    }
    pendingSources.push(source)
    pendingItemCount += entryCount
  })
  flushPending()
  return batches
}

const normalizeOptionalNonNegativeInteger = (value: unknown): number | undefined | null => {
  if (value === '' || value == null) return undefined
  const numericValue = typeof value === 'number' ? value : Number(value)
  if (!Number.isInteger(numericValue) || numericValue < 0) return null
  return numericValue
}

const collectImportOptions = (): CodexImportOptions | null => {
  const normalizedContent = content.value.trim()
  const sourceContents = fileContents.value.map((item) => item.content.trim()).filter(Boolean)
  if (!normalizedContent && sourceContents.length === 0) {
    appStore.showError(t('admin.accounts.codexSessionImportEmpty'))
    return null
  }

  const normalizedConcurrency = normalizeOptionalNonNegativeInteger(concurrency.value)
  if (normalizedConcurrency === null) {
    appStore.showError(t('admin.accounts.codexSessionImportInvalidConcurrency'))
    return null
  }
  const normalizedPriority = normalizeOptionalNonNegativeInteger(priority.value)
  if (normalizedPriority === null) {
    appStore.showError(t('admin.accounts.codexSessionImportInvalidPriority'))
    return null
  }

  return {
    sources: [
      ...(normalizedContent ? [{ content: normalizedContent, kind: 'pasted' as const }] : []),
      ...sourceContents.map((sourceContent) => ({ content: sourceContent, kind: 'file' as const }))
    ],
    concurrency: normalizedConcurrency,
    priority: normalizedPriority,
    name: name.value.trim() || undefined,
    groupIds: [...groupIds.value],
    updateExisting: updateExisting.value
  }
}

const buildSourcePayload = (sources: CodexImportSource[]): Pick<CodexSessionImportRequest, 'content' | 'contents'> => {
  const pastedSource = sources.find((source) => source.kind === 'pasted')
  const fileSources = sources.filter((source) => source.kind === 'file').map((source) => source.content)
  return {
    ...(pastedSource ? { content: pastedSource.content } : {}),
    ...(fileSources.length > 0 ? { contents: fileSources } : {})
  }
}

const setImportProgress = (
  phase: 'preview' | 'import',
  batchIndex: number,
  totalBatches: number,
  processedSources: number,
  totalSources: number
) => {
  importProgress.value = {
    phase,
    current: batchIndex + 1,
    total: totalBatches,
    processed: processedSources,
    total_items: totalSources,
    percent: totalSources > 0 ? Math.floor((processedSources / totalSources) * 100) : 100
  }
}

const previewImportSources = async (
  sources: CodexImportSource[],
  generation: number
): Promise<CodexSessionImportPreviewResult> => {
  const batches = chunkSources(sources)
  const aggregate: CodexSessionImportPreviewResult = {
    total: 0,
    source_entry_counts: [],
    http_status_401: []
  }
  let processedSources = 0

  for (let batchIndex = 0; batchIndex < batches.length; batchIndex += 1) {
    const batch = batches[batchIndex]
    setImportProgress('preview', batchIndex, batches.length, processedSources, sources.length)
    const batchResult = await adminAPI.accounts.previewCodexSessionImport({
      ...buildSourcePayload(batch),
      index_offset: aggregate.total
    })
    if (generation !== sourceGeneration || !props.show) {
      throw stalePreview
    }
    aggregate.total += batchResult.total || 0
    aggregate.source_entry_counts.push(...(batchResult.source_entry_counts || []))
    aggregate.http_status_401.push(...(batchResult.http_status_401 || []))
    processedSources += batch.length
    setImportProgress('preview', batchIndex, batches.length, processedSources, sources.length)
  }

  return aggregate
}

const executeImport = async (
  options: CodexImportOptions,
  preview: CodexSessionImportPreviewResult,
  filterHTTPStatus401: boolean
) => {
  const batches = buildImportBatches(options.sources, preview.source_entry_counts)
  const plannedTotal = batches.reduce((total, batch) => total + batch.itemCount, 0)
  if (plannedTotal !== preview.total) {
    throw new Error(t('admin.accounts.codexSessionImportPreviewMismatch'))
  }
  if (batches.length === 0) {
    throw new Error(t('admin.accounts.codexSessionImportEmpty'))
  }
  const aggregate = createEmptyResult()
  let processedItems = 0
  let entryOffset = 0

  result.value = aggregate
  importProgress.value = null
  for (let batchIndex = 0; batchIndex < batches.length; batchIndex += 1) {
    const batch = batches[batchIndex]
    setImportProgress('import', batchIndex, batches.length, processedItems, preview.total)

    const payload: CodexSessionImportRequest = {
      ...buildSourcePayload(batch.sources),
      ...(batch.entryStart === undefined ? {} : { entry_start: batch.entryStart }),
      ...(batch.entryLimit === undefined ? {} : { entry_limit: batch.entryLimit }),
      name: options.name,
      group_ids: [...options.groupIds],
      ...(options.concurrency === undefined ? {} : { concurrency: options.concurrency }),
      ...(options.priority === undefined ? {} : { priority: options.priority }),
      index_offset: entryOffset,
      total_items: preview.total,
      filter_http_status_401: filterHTTPStatus401,
      update_existing: options.updateExisting,
      skip_default_group_bind: true,
      extra: {
        import_ui: 'codex_session_compat'
      }
    }
    const batchResult = await adminAPI.accounts.importCodexSession(payload)
    if (batchResult.total !== batch.itemCount) {
      throw new Error(t('admin.accounts.codexSessionImportPreviewMismatch'))
    }
    appendImportResult(aggregate, batchResult)
    entryOffset += batchResult.total || 0
    processedItems += batchResult.total || 0
    result.value = cloneImportResult(aggregate)
    setImportProgress('import', batchIndex, batches.length, processedItems, preview.total)
  }
  if (aggregate.total !== preview.total) {
    throw new Error(t('admin.accounts.codexSessionImportPreviewMismatch'))
  }
  importProgress.value = null

  const msgParams = {
    created: aggregate.created,
    updated: aggregate.updated,
    skipped: aggregate.skipped,
    failed: aggregate.failed
  }
  if (aggregate.failed > 0) {
    appStore.showError(t('admin.accounts.codexSessionImportCompletedWithErrors', msgParams))
  } else {
    appStore.showSuccess(t('admin.accounts.codexSessionImportSuccess', msgParams))
    emit('imported')
  }
}

const runImport = async (
  options: CodexImportOptions,
  preview: CodexSessionImportPreviewResult,
  filterHTTPStatus401: boolean
) => {
  importing.value = true
  try {
    await executeImport(options, preview, filterHTTPStatus401)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.codexSessionImportFailed'))
  } finally {
    importing.value = false
  }
}

const handleImport = async () => {
  if (importing.value || loadingFiles.value) return
  const options = collectImportOptions()
  if (!options) return

  importing.value = true
  const generation = ++sourceGeneration
  result.value = null
  previewResult.value = null
  try {
    const preview = await previewImportSources(options.sources, generation)
    if (generation !== sourceGeneration || !props.show) return
    if (preview.http_status_401.length > 0) {
      previewResult.value = preview
      importProgress.value = null
      return
    }
    await executeImport(options, preview, false)
  } catch (error: any) {
    if (error !== stalePreview) {
      appStore.showError(error?.message || t('admin.accounts.codexSessionImportFailed'))
    }
  } finally {
    importProgress.value = null
    importing.value = false
  }
}

const handleHTTP401Decision = async (filterHTTPStatus401: boolean) => {
  if (importing.value || loadingFiles.value) return
  const options = collectImportOptions()
  const preview = previewResult.value
  if (!options || !preview) return
  previewResult.value = null
  sourceGeneration += 1
  await runImport(options, preview, filterHTTPStatus401)
}

</script>
