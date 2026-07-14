import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('OpenAI Fast/Flex policy locale keys', () => {
  it('exposes user scope copy at the runtime zh path', () => {
    expect(zh.admin.settings.openaiFastPolicy).toMatchObject({
      userIds: '指定用户',
      userSearchPlaceholder: '输入用户邮箱搜索',
      userSearchEmpty: '未找到匹配用户',
      userDeleted: '（已删除）',
      userIdFallback: '用户 #{id}',
      removeUser: '移除用户'
    })
  })

  it('exposes user scope copy at the runtime en path', () => {
    expect(en.admin.settings.openaiFastPolicy).toMatchObject({
      userIds: 'Specific users',
      userSearchPlaceholder: 'Search by user email',
      userSearchEmpty: 'No matching users found',
      userDeleted: '(deleted)',
      userIdFallback: 'User #{id}',
      removeUser: 'Remove user'
    })
  })
})
