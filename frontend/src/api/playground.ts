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

export interface VideoJobRequest {
  baseUrl: string
  apiKey: string
  model: string
  prompt: string
  seconds: number
  size: string
  resolutionName: string
  preset: string
  signal?: AbortSignal
}

export interface VideoJob {
  id: string
  object: string
  created_at: number
  status: 'queued' | 'running' | 'completed' | 'failed' | string
  model: string
  progress: number
  prompt: string
  seconds: string
  size: string
  quality: string
  completed_at?: number
  error?: {
    code?: string
    message?: string
  } | Record<string, unknown>
}

interface VideoJobResponse extends VideoJob {
  error?: VideoJob['error'] & {
    message?: string
  }
}

export interface VideoJobLookupRequest {
  baseUrl: string
  apiKey: string
  id: string
  signal?: AbortSignal
}

export interface AudioSpeechRequest {
  baseUrl: string
  apiKey: string
  model: string
  input: string
  voice: string
  format: string
  signal?: AbortSignal
}

export interface AudioFileRequest {
  baseUrl: string
  apiKey: string
  model: string
  file: File
  language?: string
  signal?: AbortSignal
}

export interface AudioSpeechResult {
  url: string
  blob: Blob
  contentType: string
}

interface AudioTextResponse {
  text?: string
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

async function readGatewayError(res: Response): Promise<string> {
  const text = await res.text().catch(() => '')
  if (!text) return `${res.status} ${res.statusText}`
  try {
    const payload = JSON.parse(text) as { error?: { message?: string }; message?: string }
    return payload.error?.message || payload.message || text
  } catch {
    return text
  }
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

export async function createVideoJob(request: VideoJobRequest): Promise<VideoJob> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const res = await fetch(`${baseUrl}/v1/videos`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`,
    },
    body: JSON.stringify({
      model: request.model,
      prompt: request.prompt,
      seconds: request.seconds,
      size: request.size,
      resolution_name: request.resolutionName,
      preset: request.preset,
    }),
    signal: request.signal,
  })

  const payload = (await res.json().catch(() => ({}))) as VideoJobResponse
  if (!res.ok) {
    throw new Error(payload.error?.message || `${res.status} ${res.statusText}`)
  }
  if (!payload.id) {
    throw new Error('Empty video job response from gateway')
  }
  return payload
}

export async function getVideoJob(request: VideoJobLookupRequest): Promise<VideoJob> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const res = await fetch(`${baseUrl}/v1/videos/${encodeURIComponent(request.id)}`, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${request.apiKey}`,
    },
    signal: request.signal,
  })

  const payload = (await res.json().catch(() => ({}))) as VideoJobResponse
  if (!res.ok) {
    throw new Error(payload.error?.message || `${res.status} ${res.statusText}`)
  }
  if (!payload.id) {
    throw new Error('Empty video job response from gateway')
  }
  return payload
}

export function videoContentUrl(baseUrl: string, id: string): string {
  return `${normalizeGatewayBaseUrl(baseUrl)}/v1/videos/${encodeURIComponent(id)}/content`
}

export async function getVideoContentObjectUrl(request: VideoJobLookupRequest): Promise<{ url: string; blob: Blob; contentType: string }> {
  const res = await fetch(videoContentUrl(request.baseUrl, request.id), {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${request.apiKey}`,
    },
    signal: request.signal,
  })
  if (!res.ok) {
    throw new Error(await readGatewayError(res))
  }
  const blob = await res.blob()
  return {
    url: URL.createObjectURL(blob),
    blob,
    contentType: res.headers.get('Content-Type') || blob.type || 'video/mp4',
  }
}

export async function createAudioSpeech(request: AudioSpeechRequest): Promise<AudioSpeechResult> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const res = await fetch(`${baseUrl}/v1/audio/speech`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${request.apiKey}`,
    },
    body: JSON.stringify({
      model: request.model,
      input: request.input,
      voice: request.voice,
      response_format: request.format,
    }),
    signal: request.signal,
  })
  if (!res.ok) {
    throw new Error(await readGatewayError(res))
  }
  const blob = await res.blob()
  return {
    url: URL.createObjectURL(blob),
    blob,
    contentType: res.headers.get('Content-Type') || blob.type || 'audio/mpeg',
  }
}

async function createAudioText(endpoint: 'transcriptions' | 'translations', request: AudioFileRequest): Promise<string> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const form = new FormData()
  form.set('model', request.model)
  form.set('file', request.file)
  if (request.language) {
    form.set('language', request.language)
  }

  const res = await fetch(`${baseUrl}/v1/audio/${endpoint}`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${request.apiKey}`,
    },
    body: form,
    signal: request.signal,
  })

  const text = await res.text()
  let payload: AudioTextResponse | null = null
  try {
    payload = JSON.parse(text) as AudioTextResponse
  } catch {
    payload = null
  }
  if (!res.ok) {
    throw new Error(payload?.error?.message || payload?.text || text || `${res.status} ${res.statusText}`)
  }
  return payload?.text || text
}

export function createAudioTranscription(request: AudioFileRequest): Promise<string> {
  return createAudioText('transcriptions', request)
}

export function createAudioTranslation(request: AudioFileRequest): Promise<string> {
  return createAudioText('translations', request)
}

export function resolveGatewayBaseUrl(configuredBaseUrl: string | undefined): string {
  return normalizeGatewayBaseUrl(configuredBaseUrl || '')
}
