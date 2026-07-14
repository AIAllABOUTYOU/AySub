import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CodexSessionImportModal from '../CodexSessionImportModal.vue'
import type { CodexSessionImportResult } from '@/types'

const { importCodexSession, previewCodexSessionImport, readImportTextFiles, showError, showSuccess } = vi.hoisted(() => ({
  importCodexSession: vi.fn(),
  previewCodexSessionImport: vi.fn(),
  readImportTextFiles: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      importCodexSession,
      previewCodexSessionImport
    }
  }
}))

vi.mock('@/utils/importTextFiles', () => ({
  readImportTextFiles
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.accounts.codexSessionImportResultSummary') {
          return `result:${params?.total}/${params?.created}/${params?.updated}/${params?.skipped}/${params?.failed}`
        }
        if (key === 'admin.accounts.codexSessionImportSuccess') {
          return `success:${params?.created}/${params?.updated}/${params?.skipped}`
        }
        return key
      }
    })
  }
})

const groups = [
  { id: 7, name: 'OpenAI A', platform: 'openai' },
  { id: 9, name: 'OpenAI B', platform: 'openai' }
]

const BaseDialogStub = {
  template: '<div><slot /><slot name="footer" /></div>'
}

const GroupSelectorStub = {
  name: 'GroupSelector',
  props: ['modelValue', 'groups'],
  emits: ['update:modelValue'],
  template: '<div data-testid="group-selector"></div>'
}

function mountModal() {
  return mount(CodexSessionImportModal, {
    props: {
      show: true,
      groups
    } as any,
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        GroupSelector: GroupSelectorStub
      }
    }
  })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function importResult(overrides: Partial<CodexSessionImportResult>): CodexSessionImportResult {
  return {
    total: 0,
    created: 0,
    updated: 0,
    skipped: 0,
    failed: 0,
    items: [],
    warnings: [],
    errors: [],
    ...overrides
  }
}

async function loadFileSources(wrapper: VueWrapper, count: number) {
  const loaded = Array.from({ length: count }, (_, index) => ({
    name: `account-${index + 1}.json`,
    content: `session-${index + 1}`
  }))
  readImportTextFiles.mockResolvedValueOnce({ loaded, skipped: [] })

  const input = wrapper.findAll('input[type="file"]')[0]
  const files = [new File(['ignored'], 'accounts.zip', { type: 'application/zip' })]
  Object.defineProperty(input.element, 'files', {
    configurable: true,
    value: files
  })
  await input.trigger('change')
  await flushPromises()

  expect(readImportTextFiles).toHaveBeenCalledWith(files, expect.any(Object))
}

function jsonPayload(callIndex: number) {
  return JSON.parse(JSON.stringify(importCodexSession.mock.calls[callIndex][0])) as Record<string, any>
}

function jsonPreviewPayload(callIndex: number) {
  return JSON.parse(JSON.stringify(previewCodexSessionImport.mock.calls[callIndex][0])) as Record<string, any>
}

describe('CodexSessionImportModal', () => {
  beforeEach(() => {
    importCodexSession.mockReset()
    previewCodexSessionImport.mockReset()
    readImportTextFiles.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('将 121 个文件来源分批预检，再按账号条目数严格串行导入', async () => {
    const firstPreview = deferred<{ total: number, source_entry_counts: number[], http_status_401: [] }>()
    const secondPreview = deferred<{ total: number, source_entry_counts: number[], http_status_401: [] }>()
    const thirdPreview = deferred<{ total: number, source_entry_counts: number[], http_status_401: [] }>()
    previewCodexSessionImport
      .mockReturnValueOnce(firstPreview.promise)
      .mockReturnValueOnce(secondPreview.promise)
      .mockReturnValueOnce(thirdPreview.promise)
    const first = deferred<CodexSessionImportResult>()
    const second = deferred<CodexSessionImportResult>()
    const third = deferred<CodexSessionImportResult>()
    importCodexSession
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)
      .mockReturnValueOnce(third.promise)

    const wrapper = mountModal()
    await loadFileSources(wrapper, 121)

    const groupSelector = wrapper.getComponent({ name: 'GroupSelector' })
    groupSelector.vm.$emit('update:modelValue', [7, 9])
    await wrapper.vm.$nextTick()

    const [concurrencyInput, priorityInput] = wrapper.findAll('input[type="number"]')
    await concurrencyInput.setValue('')
    await priorityInput.setValue('')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(1)
    expect(jsonPreviewPayload(0)).toEqual({
      contents: Array.from({ length: 50 }, (_, index) => `session-${index + 1}`),
      index_offset: 0
    })
    expect(importCodexSession).not.toHaveBeenCalled()

    firstPreview.resolve({
      total: 55,
      source_entry_counts: [6, ...Array.from({ length: 49 }, () => 1)],
      http_status_401: []
    })
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(2)
    expect(jsonPreviewPayload(1)).toEqual({
      contents: Array.from({ length: 50 }, (_, index) => `session-${index + 51}`),
      index_offset: 55
    })
    expect(importCodexSession).not.toHaveBeenCalled()

    secondPreview.resolve({
      total: 60,
      source_entry_counts: [11, ...Array.from({ length: 49 }, () => 1)],
      http_status_401: []
    })
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(3)
    expect(jsonPreviewPayload(2)).toEqual({
      contents: Array.from({ length: 21 }, (_, index) => `session-${index + 101}`),
      index_offset: 115
    })
    expect(importCodexSession).not.toHaveBeenCalled()

    thirdPreview.resolve({
      total: 21,
      source_entry_counts: Array.from({ length: 21 }, () => 1),
      http_status_401: []
    })
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(3)
    expect(importCodexSession).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('imported')).toBeUndefined()
    expect(jsonPayload(0)).toMatchObject({
      contents: Array.from({ length: 45 }, (_, index) => `session-${index + 1}`),
      group_ids: [7, 9],
      index_offset: 0,
      total_items: 136,
      filter_http_status_401: false
    })
    expect(jsonPayload(0)).not.toHaveProperty('concurrency')
    expect(jsonPayload(0)).not.toHaveProperty('priority')

    first.resolve(importResult({
      total: 50,
      created: 40,
      updated: 5,
      skipped: 5,
      items: [{ index: 1, action: 'created', account_id: 101 }],
      warnings: [{ index: 2, message: 'first warning' }]
    }))
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(2)
    expect(wrapper.emitted('imported')).toBeUndefined()
    expect(jsonPayload(1)).toMatchObject({
      contents: Array.from({ length: 40 }, (_, index) => `session-${index + 46}`),
      group_ids: [7, 9],
      index_offset: 50,
      total_items: 136,
      filter_http_status_401: false
    })

    second.resolve(importResult({
      total: 50,
      created: 30,
      updated: 10,
      skipped: 10,
      items: [{ index: 51, action: 'updated', account_id: 151 }],
      warnings: [{ index: 52, message: 'second warning' }]
    }))
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(3)
    expect(wrapper.emitted('imported')).toBeUndefined()
    expect(jsonPayload(2)).toMatchObject({
      contents: Array.from({ length: 36 }, (_, index) => `session-${index + 86}`),
      group_ids: [7, 9],
      index_offset: 100,
      total_items: 136,
      filter_http_status_401: false
    })

    third.resolve(importResult({
      total: 36,
      created: 36,
      items: [{ index: 101, action: 'created', account_id: 201 }]
    }))
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(3)
    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(wrapper.text()).toContain('result:136/106/15/15/0')
    expect(showSuccess).toHaveBeenCalledWith('success:106/15/15')
    expect(showError).not.toHaveBeenCalled()
  })

  it('显式输入 0 时保留 concurrency 和 priority', async () => {
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 1,
      source_entry_counts: [1],
      http_status_401: []
    })
    importCodexSession.mockResolvedValueOnce(importResult({ total: 1, created: 1 }))

    const wrapper = mountModal()
    await wrapper.get('textarea').setValue('single-session')
    const [concurrencyInput, priorityInput] = wrapper.findAll('input[type="number"]')
    await concurrencyInput.setValue('0')
    await priorityInput.setValue('0')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(1)
    expect(jsonPayload(0)).toMatchObject({
      content: 'single-session',
      concurrency: 0,
      priority: 0,
      group_ids: [],
      index_offset: 0,
      total_items: 1,
      filter_http_status_401: false
    })
    expect(wrapper.emitted('imported')).toHaveLength(1)
  })

  it('预检发现 HTTP 401 时先展示清单，点击过滤后跳过这些来源再导入', async () => {
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 2,
      source_entry_counts: [2],
      http_status_401: [{
        index: 2,
        name: 'Unauthorized account',
        email: 'unauthorized@example.com',
        account_id: 'acct-401',
        http_status: 401
      }]
    })
    importCodexSession.mockResolvedValueOnce(importResult({
      total: 2,
      created: 1,
      skipped: 1
    }))

    const wrapper = mountModal()
    await wrapper.get('textarea').setValue('[{"access_token":"valid"},{"http_status":401}]')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(1)
    expect(importCodexSession).not.toHaveBeenCalled()
    const review = wrapper.get('[data-testid="codex-import-http-401-review"]')
    expect(review.text()).toContain('#2 Unauthorized account - HTTP 401')
    expect(wrapper.emitted('imported')).toBeUndefined()

    await wrapper.get('[data-testid="codex-import-filter-401"]').trigger('click')
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(1)
    expect(jsonPayload(0)).toMatchObject({
      content: '[{"access_token":"valid"},{"http_status":401}]',
      index_offset: 0,
      total_items: 2,
      filter_http_status_401: true
    })
    expect(wrapper.emitted('imported')).toHaveLength(1)
    expect(showSuccess).toHaveBeenCalledWith('success:1/0/1')
  })

  it('预检发现 HTTP 401 时点击仍导入会发送 filter_http_status_401=false', async () => {
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 1,
      source_entry_counts: [1],
      http_status_401: [{
        index: 1,
        name: 'Keep account',
        http_status: 401
      }]
    })
    importCodexSession.mockResolvedValueOnce(importResult({ total: 1, created: 1 }))

    const wrapper = mountModal()
    await wrapper.get('textarea').setValue('{"access_token":"keep","http_status":401}')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(importCodexSession).not.toHaveBeenCalled()
    await wrapper.get('[data-testid="codex-import-include-401"]').trigger('click')
    await flushPromises()

    expect(importCodexSession).toHaveBeenCalledTimes(1)
    expect(jsonPayload(0)).toMatchObject({
      content: '{"access_token":"keep","http_status":401}',
      index_offset: 0,
      total_items: 1,
      filter_http_status_401: false
    })
    expect(wrapper.emitted('imported')).toHaveLength(1)
  })

  it('单个来源包含 121 个账号时按 entry_start 和 entry_limit 分三批导入', async () => {
    const sourceContent = '[{"access_token":"account-1"},"...121 accounts..."]'
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 121,
      source_entry_counts: [121],
      http_status_401: []
    })
    importCodexSession
      .mockResolvedValueOnce(importResult({ total: 50, created: 50 }))
      .mockResolvedValueOnce(importResult({ total: 50, created: 50 }))
      .mockResolvedValueOnce(importResult({ total: 21, created: 21 }))

    const wrapper = mountModal()
    await wrapper.get('textarea').setValue(sourceContent)
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(1)
    expect(jsonPreviewPayload(0)).toEqual({
      content: sourceContent,
      index_offset: 0
    })
    expect(importCodexSession).toHaveBeenCalledTimes(3)
    expect(jsonPayload(0)).toMatchObject({
      content: sourceContent,
      entry_start: 0,
      entry_limit: 50,
      index_offset: 0,
      total_items: 121
    })
    expect(jsonPayload(1)).toMatchObject({
      content: sourceContent,
      entry_start: 50,
      entry_limit: 50,
      index_offset: 50,
      total_items: 121
    })
    expect(jsonPayload(2)).toMatchObject({
      content: sourceContent,
      entry_start: 100,
      entry_limit: 21,
      index_offset: 100,
      total_items: 121
    })
    expect(wrapper.text()).toContain('result:121/121/0/0/0')
    expect(wrapper.emitted('imported')).toHaveLength(1)
  })

  it('预检未完成时修改粘贴内容会丢弃旧预检结果并清理进度', async () => {
    const pendingPreview = deferred<{
      total: number
      source_entry_counts: number[]
      http_status_401: Array<{ index: number, name: string, http_status: number }>
    }>()
    previewCodexSessionImport.mockReturnValueOnce(pendingPreview.promise)

    const wrapper = mountModal()
    const textarea = wrapper.get('textarea')
    await textarea.setValue('{"access_token":"old"}')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('admin.accounts.codexSessionImportPreviewProgress')
    await textarea.setValue('{"access_token":"new"}')

    pendingPreview.resolve({
      total: 1,
      source_entry_counts: [1],
      http_status_401: [{ index: 1, name: 'Old unauthorized account', http_status: 401 }]
    })
    await flushPromises()

    expect(importCodexSession).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="codex-import-http-401-review"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.accounts.codexSessionImportPreviewProgress')
    expect(wrapper.emitted('imported')).toBeUndefined()
    expect(showError).not.toHaveBeenCalled()
  })

  it('预检 total 与 source_entry_counts 总和不一致时阻止正式导入', async () => {
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 51,
      source_entry_counts: [50],
      http_status_401: []
    })

    const wrapper = mountModal()
    await wrapper.get('textarea').setValue('{"access_token":"mismatch"}')
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(previewCodexSessionImport).toHaveBeenCalledTimes(1)
    expect(importCodexSession).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.accounts.codexSessionImportPreviewMismatch')
    expect(wrapper.emitted('imported')).toBeUndefined()
  })

  it('较早的文件读取晚完成时不会覆盖较新的文件选择', async () => {
    const firstRead = deferred<{
      loaded: Array<{ name: string, content: string }>
      skipped: []
    }>()
    const secondRead = deferred<{
      loaded: Array<{ name: string, content: string }>
      skipped: []
    }>()
    readImportTextFiles
      .mockReturnValueOnce(firstRead.promise)
      .mockReturnValueOnce(secondRead.promise)
    previewCodexSessionImport.mockResolvedValueOnce({
      total: 1,
      source_entry_counts: [1],
      http_status_401: []
    })
    importCodexSession.mockResolvedValueOnce(importResult({ total: 1, created: 1 }))

    const wrapper = mountModal()
    const input = wrapper.findAll('input[type="file"]')[0]
    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [new File(['old'], 'old.json', { type: 'application/json' })]
    })
    await input.trigger('change')
    await flushPromises()

    Object.defineProperty(input.element, 'files', {
      configurable: true,
      value: [new File(['new'], 'new.json', { type: 'application/json' })]
    })
    await input.trigger('change')
    await flushPromises()
    expect(readImportTextFiles).toHaveBeenCalledTimes(2)

    secondRead.resolve({
      loaded: [{ name: 'new.json', content: 'new-session' }],
      skipped: []
    })
    await flushPromises()
    firstRead.resolve({
      loaded: [{ name: 'old.json', content: 'old-session' }],
      skipped: []
    })
    await flushPromises()

    expect(showSuccess).toHaveBeenCalledTimes(1)
    await wrapper.get('#codex-session-import-form').trigger('submit')
    await flushPromises()

    expect(jsonPreviewPayload(0)).toEqual({
      contents: ['new-session'],
      index_offset: 0
    })
    expect(jsonPayload(0)).toMatchObject({ contents: ['new-session'] })
  })
})
