export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export interface ChatCompletionRequest {
  baseUrl: string
  apiKey: string
  model: string
  messages: ChatMessage[]
  temperature: number
  signal?: AbortSignal
}

interface ChatCompletionResponse {
  choices?: Array<{
    message?: {
      content?: string | Array<{ type?: string; text?: string }>
    }
    text?: string
  }>
  output_text?: string
  error?: {
    message?: string
  }
}

export interface ImageGenerationRequest {
  baseUrl: string
  apiKey: string
  model: string
  prompt: string
  size: string
  signal?: AbortSignal
}

export interface GeneratedImage {
  url: string
  revisedPrompt?: string
}

interface ImageGenerationResponse {
  data?: Array<{
    url?: string
    b64_json?: string
    revised_prompt?: string
  }>
  error?: {
    message?: string
  }
}

function normalizeGatewayBaseUrl(baseUrl: string): string {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')
  if (!trimmed) return window.location.origin
  if (trimmed.endsWith('/v1')) return trimmed.slice(0, -3)
  return trimmed
}

function extractText(response: ChatCompletionResponse): string {
  if (response.output_text) return response.output_text

  const choice = response.choices?.[0]
  if (!choice) return ''
  if (choice.text) return choice.text

  const content = choice.message?.content
  if (typeof content === 'string') return content
  if (Array.isArray(content)) {
    return content
      .map((part) => part.text || '')
      .filter(Boolean)
      .join('\n')
  }
  return ''
}

export async function createChatCompletion(request: ChatCompletionRequest): Promise<string> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const res = await fetch(`${baseUrl}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`,
    },
    body: JSON.stringify({
      model: request.model,
      messages: request.messages,
      temperature: request.temperature,
      stream: false,
    }),
    signal: request.signal,
  })

  const payload = (await res.json().catch(() => ({}))) as ChatCompletionResponse
  if (!res.ok) {
    throw new Error(payload.error?.message || `${res.status} ${res.statusText}`)
  }

  const text = extractText(payload)
  if (!text) {
    throw new Error('Empty response from gateway')
  }
  return text
}

export async function createImageGeneration(request: ImageGenerationRequest): Promise<GeneratedImage[]> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const res = await fetch(`${baseUrl}/v1/images/generations`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`,
    },
    body: JSON.stringify({
      model: request.model,
      prompt: request.prompt,
      size: request.size,
      n: 1,
      response_format: 'url',
    }),
    signal: request.signal,
  })

  const payload = (await res.json().catch(() => ({}))) as ImageGenerationResponse
  if (!res.ok) {
    throw new Error(payload.error?.message || `${res.status} ${res.statusText}`)
  }

  const images = (payload.data || [])
    .map((item): GeneratedImage | null => {
      const url = item.url || (item.b64_json ? `data:image/png;base64,${item.b64_json}` : '')
      if (!url) return null
      return {
        url,
        revisedPrompt: item.revised_prompt,
      }
    })
    .filter((item): item is GeneratedImage => item !== null)

  if (images.length === 0) {
    throw new Error('Empty image response from gateway')
  }
  return images
}

export function resolveGatewayBaseUrl(configuredBaseUrl: string | undefined): string {
  return normalizeGatewayBaseUrl(configuredBaseUrl || '')
}
