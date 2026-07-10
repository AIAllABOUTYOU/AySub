import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post },
}))

import { getRollbackVersions, rollback } from '@/api/admin/system'

describe('admin system rollback api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('loads allowed rollback versions', async () => {
    const response = { versions: [{ version: '0.1.138', published_at: '2026-07-01', html_url: '#' }] }
    get.mockResolvedValue({ data: response })

    await expect(getRollbackVersions()).resolves.toEqual(response)
    expect(get).toHaveBeenCalledWith('/admin/system/rollback-versions')
  })

  it('posts a target version for online rollback', async () => {
    post.mockResolvedValue({ data: { message: 'ok', need_restart: true } })

    await rollback('0.1.138')

    expect(post).toHaveBeenCalledWith('/admin/system/rollback', { version: '0.1.138' })
  })

  it('keeps the no-body local backup rollback contract', async () => {
    post.mockResolvedValue({ data: { message: 'ok', need_restart: true } })

    await rollback()

    expect(post).toHaveBeenCalledWith('/admin/system/rollback', undefined)
  })
})
