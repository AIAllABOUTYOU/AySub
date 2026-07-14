import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const createSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf8'
)
const editSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/EditAccountModal.vue'),
  'utf8'
)
const reauthSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/ReAuthAccountModal.vue'),
  'utf8'
)
const grokApiSource = readFileSync(resolve(process.cwd(), 'src/api/admin/grok.ts'), 'utf8')
const grokOAuthSource = readFileSync(
  resolve(process.cwd(), 'src/composables/useGrokOAuth.ts'),
  'utf8'
)

describe('xAI account defaults', () => {
  it('keeps the AySub xai platform and backend-compatible API Key base URL', () => {
    expect(createSource).toContain("form.platform = 'xai'")
    expect(createSource).toContain('data-testid="xai-account-type-api-key"')
    expect(createSource).toContain("? 'https://api.x.ai/v1'")
    expect(editSource).toContain("account.platform === 'xai'")
    expect(editSource).toContain("return 'https://api.x.ai/v1'")
  })

  it('offers OAuth without replacing the existing API Key and Cookie account types', () => {
    expect(createSource).toContain('data-testid="xai-account-type-oauth"')
    expect(createSource).toContain("accountCategory = 'apikey'")
    expect(createSource).toContain("accountCategory = 'cookie'")
    expect(createSource).toContain("createAccountAndFinish('xai', 'cookie'")
  })

  it('uses the xAI OAuth API and Grok OAuth client for reauthorization', () => {
    expect(grokApiSource).toContain("'/admin/xai/oauth/auth-url'")
    expect(grokApiSource).toContain("'/admin/xai/oauth/exchange-code'")
    expect(reauthSource).toContain("import { useGrokOAuth }")
    expect(reauthSource).toContain('const grokOAuth = useGrokOAuth()')
    expect(reauthSource).toContain('credentials: grokOAuth.buildCredentials(tokenInfo)')
    expect(grokOAuthSource).toContain('base_url: GROK_CLI_BASE_URL')
  })
})
