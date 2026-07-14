import { beforeEach, describe, expect, it, vi } from 'vitest'
import type {
  CodexSessionImportPreviewResult,
  CodexSessionImportRequest,
  CodexSessionImportResult
} from '@/types'

const { post } = vi.hoisted(() => ({
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { post }
}))

import { importCodexSession, previewCodexSessionImport } from '@/api/admin/accounts'

const payload: CodexSessionImportRequest = {
  contents: ['session-a', 'session-b'],
  group_ids: [7],
  concurrency: 3,
  priority: 50,
  index_offset: 10
}

describe('admin accounts Codex session import api', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('posts an import request with the extended timeout and returns response data', async () => {
    const response: CodexSessionImportResult = {
      total: 2,
      created: 1,
      updated: 1,
      skipped: 0,
      failed: 0
    }
    post.mockResolvedValue({ data: response })

    await expect(importCodexSession(payload)).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/admin/accounts/import/codex-session', payload, {
      timeout: 120000
    })
  })

  it('posts a preview request with the extended timeout and returns response data', async () => {
    const response: CodexSessionImportPreviewResult = {
      total: 2,
      source_entry_counts: [1, 1],
      http_status_401: [
        {
          index: 11,
          name: 'expired-account',
          http_status: 401
        }
      ]
    }
    post.mockResolvedValue({ data: response })

    await expect(previewCodexSessionImport(payload)).resolves.toEqual(response)
    expect(post).toHaveBeenCalledWith('/admin/accounts/import/codex-session/preview', payload, {
      timeout: 120000
    })
  })
})
