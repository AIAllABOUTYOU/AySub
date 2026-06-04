import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

type LocaleTree = Record<string, unknown>

function collectMessagePaths(tree: LocaleTree, prefix = ''): string[] {
  return Object.entries(tree).flatMap(([key, value]) => {
    const path = prefix ? `${prefix}.${key}` : key
    if (typeof value === 'string') {
      return [path]
    }
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return collectMessagePaths(value as LocaleTree, path)
    }
    return []
  })
}

describe('locale message syntax', () => {
  it.each([
    ['en', en],
    ['zh', zh],
  ] as const)('does not contain raw email addresses in %s messages', (_locale, messages) => {
    const rawEmailPattern = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+/
    const pathsWithRawEmails = collectMessagePaths(messages).filter((path) => {
      const value = path.split('.').reduce<unknown>((node, key) => {
        return node && typeof node === 'object' ? (node as LocaleTree)[key] : undefined
      }, messages)

      return typeof value === 'string' && rawEmailPattern.test(value)
    })

    expect(pathsWithRawEmails).toEqual([])
  })
})
