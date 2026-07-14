import { describe, expect, it } from 'vitest'

import zh from '../locales/zh'

describe('upstream locale keys', () => {
  it('exposes dashboard overview copy at the AySub runtime path', () => {
    expect(zh.admin.dashboard).toMatchObject({
      newUsersToday: '今日新增用户',
      active: '活跃',
      ok: '正常',
      err: '错误',
      create: '创建',
      userUsageTrend: '用户使用趋势（Top 12）'
    })
  })

  it('exposes Claude Max simulation and refund copy at their runtime paths', () => {
    expect(zh.admin.groups.claudeMaxSimulation).toMatchObject({
      title: 'Claude Max 用量模拟',
      enabled: '已启用（模拟 1h 缓存）',
      disabled: '已禁用'
    })
    expect(zh.admin.settings.payment.allowUserRefund).toBe('允许用户退款')
  })
})
