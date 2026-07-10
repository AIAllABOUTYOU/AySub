import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const sources = [
  readFileSync(resolve(dir, '../AppHeader.vue'), 'utf8'),
  readFileSync(resolve(dir, '../PublicHeader.vue'), 'utf8'),
  readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8'),
  readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8'),
  readFileSync(resolve(dir, '../../../views/StatusView.vue'), 'utf8')
]

describe('configured URL sanitization', () => {
  it('sanitizes every configured doc URL and site logo binding', () => {
    for (const source of sources) {
      expect(source).toContain("import { sanitizeUrl } from '@/utils/url'")
    }
    expect(sources.join('\n')).not.toMatch(/const docUrl = computed\(\(\) => appStore\./)
    expect(sources.join('\n')).not.toMatch(/const siteLogo = computed\(\(\) => appStore\./)
  })
})
