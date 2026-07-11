export interface CustomHeaderRow {
  key: string
  value: string
}

export type CustomHeaderValidationError =
  | 'tooMany'
  | 'nameRequired'
  | 'invalidName'
  | 'blockedName'
  | 'invalidValue'
  | 'duplicateName'

const MAX_HEADER_ENTRIES = 64
const MAX_HEADER_NAME_LENGTH = 200
const MAX_HEADER_VALUE_LENGTH = 8192
const HEADER_NAME_PATTERN = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/

const BLOCKED_HEADER_NAMES = new Set([
  'host', 'content-length', 'content-type', 'transfer-encoding',
  'connection', 'keep-alive', 'proxy-authenticate', 'proxy-authorization',
  'proxy-connection', 'te', 'trailer', 'upgrade',
  'authorization', 'x-api-key', 'x-goog-api-key', 'cookie',
  'accept-encoding', 'sec-websocket-key', 'sec-websocket-version',
  'sec-websocket-extensions', 'sec-websocket-protocol', 'sec-websocket-accept',
  'session_id', 'conversation_id', 'x-codex-turn-state', 'x-codex-turn-metadata',
  'chatgpt-account-id', 'x-claude-code-session-id', 'x-client-request-id'
])

export function normalizeCustomHeaderRows(rows: CustomHeaderRow[]): {
  headers: Record<string, string>
  error: CustomHeaderValidationError | null
} {
  if (rows.length > MAX_HEADER_ENTRIES) return { headers: {}, error: 'tooMany' }

  const headers: Record<string, string> = {}
  for (const row of rows) {
    const name = row.key.trim().toLowerCase()
    const value = row.value.trim()
    if (!name && !value) continue
    if (!name) return { headers: {}, error: 'nameRequired' }
    if (name.length > MAX_HEADER_NAME_LENGTH || !HEADER_NAME_PATTERN.test(name)) {
      return { headers: {}, error: 'invalidName' }
    }
    if (BLOCKED_HEADER_NAMES.has(name)) return { headers: {}, error: 'blockedName' }
    if (value.length > MAX_HEADER_VALUE_LENGTH || /[\r\n\0]/.test(value)) {
      return { headers: {}, error: 'invalidValue' }
    }
    if (Object.prototype.hasOwnProperty.call(headers, name)) {
      return { headers: {}, error: 'duplicateName' }
    }
    headers[name] = value
  }
  return { headers, error: null }
}
