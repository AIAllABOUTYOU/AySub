import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'

import { createImageGeneration } from '../playground'

const originalCreateObjectURL = URL.createObjectURL

describe('playground image generation', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: vi.fn(() => 'blob:grok-local-image'),
    })
  })

  afterEach(() => {
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: originalCreateObjectURL,
    })
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('downloads local Grok image URLs with gateway authorization', async () => {
    const fetchMock = vi.mocked(fetch)
    fetchMock
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: [{ url: '/v1/files/image?id=abc123', revised_prompt: 'draw a cat' }],
        }),
      } as Response)
      .mockResolvedValueOnce({
        ok: true,
        blob: async () => new Blob(['png-bytes'], { type: 'image/png' }),
      } as Response)

    const images = await createImageGeneration({
      baseUrl: 'https://gateway.example/v1',
      apiKey: 'sk-test',
      model: 'grok-imagine-image',
      prompt: 'draw a cat',
      size: '1024x1024',
    })

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      'https://gateway.example/v1/images/generations',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ Authorization: 'Bearer sk-test' }),
      })
    )
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      'https://gateway.example/v1/files/image?id=abc123',
      expect.objectContaining({
        method: 'GET',
        headers: { Authorization: 'Bearer sk-test' },
      })
    )
    expect(images).toEqual([{ url: 'blob:grok-local-image', revisedPrompt: 'draw a cat' }])
  })
})
