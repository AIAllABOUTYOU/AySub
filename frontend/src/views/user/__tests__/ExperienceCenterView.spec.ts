import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ExperienceCenterView from '../ExperienceCenterView.vue'

const {
  listKeys,
  getAvailableChannels,
  fetchPublicSettings,
  showError,
  createImageGeneration,
  listPlaygroundSessions,
} = vi.hoisted(() => ({
  listKeys: vi.fn(),
  getAvailableChannels: vi.fn(),
  fetchPublicSettings: vi.fn(),
  showError: vi.fn(),
  createImageGeneration: vi.fn(),
  listPlaygroundSessions: vi.fn(),
}))

vi.mock('@/api/keys', () => ({
  default: {
    list: listKeys,
  },
}))

vi.mock('@/api/channels', () => ({
  default: {
    getAvailable: getAvailableChannels,
  },
}))

vi.mock('@/api/playground', () => ({
  appendPlaygroundMessage: vi.fn(),
  createPlaygroundSession: vi.fn(),
  createChatCompletion: vi.fn(),
  createAudioSpeech: vi.fn(),
  createAudioTranscription: vi.fn(),
  createAudioTranslation: vi.fn(),
  createImageGeneration,
  createVideoJob: vi.fn(),
  deletePlaygroundSession: vi.fn(),
  getPlaygroundSession: vi.fn(),
  getVideoContentObjectUrl: vi.fn(),
  getVideoJob: vi.fn(),
  listPlaygroundSessions,
  resolveGatewayBaseUrl: (value: string) => value.replace(/\/v1$/, ''),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    apiBaseUrl: 'https://fallback.example',
    fetchPublicSettings,
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          'common.refresh': 'Refresh',
          'experienceCenter.tabs.chat': 'Chat',
          'experienceCenter.tabs.image': 'Image',
          'experienceCenter.tabs.video': 'Video',
          'experienceCenter.tabs.audio': 'Audio',
          'experienceCenter.controls.apiKey': 'API key',
          'experienceCenter.controls.selectKey': 'Select key',
          'experienceCenter.controls.model': 'Model',
          'experienceCenter.controls.selectModel': 'Select model',
          'experienceCenter.controls.baseUrl': 'Base URL',
          'experienceCenter.controls.temperature': 'Temperature',
          'experienceCenter.controls.imageSize': 'Image size',
          'experienceCenter.controls.keyHint': 'Key hint',
          'experienceCenter.actions.generate': 'Generate',
          'experienceCenter.actions.generating': 'Generating',
          'experienceCenter.image.placeholder': 'Prompt',
          'experienceCenter.image.empty': 'No images',
          'experienceCenter.chat.empty': 'No messages',
          'experienceCenter.chat.placeholder': 'Message',
          'experienceCenter.sessions.title': 'Sessions',
          'experienceCenter.sessions.new': 'New',
          'experienceCenter.sessions.empty': 'No sessions',
          'keys.noGroup': 'No group',
        }
        return messages[key] ?? key
      },
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const IconStub = { template: '<span />' }

function pricing(billingMode: string) {
  return {
    billing_mode: billingMode,
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
  }
}

describe('ExperienceCenterView', () => {
  beforeEach(() => {
    listKeys.mockReset()
    getAvailableChannels.mockReset()
    fetchPublicSettings.mockReset()
    showError.mockReset()
    createImageGeneration.mockReset()
    listPlaygroundSessions.mockReset()

    listKeys.mockResolvedValue({
      items: [
        {
          id: 1,
          key: 'sk-grok',
          name: 'Grok key',
          status: 'active',
          group: { id: 1, name: 'Grok group', platform: 'xai' },
        },
      ],
    })
    getAvailableChannels.mockResolvedValue([
      {
        name: 'Grok',
        description: '',
        platforms: [
          {
            platform: 'xai',
            groups: [],
            supported_models: [
              { name: 'grok-4.20-auto', platform: 'xai', pricing: pricing('token') },
              { name: 'grok-imagine-image', platform: 'xai', pricing: pricing('image') },
              { name: 'grok-imagine-video', platform: 'xai', pricing: pricing('per_request') },
            ],
          },
        ],
      },
    ])
    fetchPublicSettings.mockResolvedValue({ api_base_url: 'https://gateway.example/v1' })
    listPlaygroundSessions.mockResolvedValue({ items: [] })
    createImageGeneration.mockResolvedValue([{ url: 'https://assets.example/image.png' }])
  })

  it('switches Grok image testing to an image-capable model', async () => {
    const wrapper = mount(ExperienceCenterView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          Icon: IconStub,
        },
      },
    })

    await flushPromises()

    expect((wrapper.vm as any).selectedModel).toBe('grok-4.20-auto')

    await wrapper.get('button:nth-of-type(2)').trigger('click')
    await flushPromises()

    expect((wrapper.vm as any).selectedModel).toBe('grok-imagine-image')

    await wrapper.get('textarea').setValue('draw a cat')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createImageGeneration).toHaveBeenCalledWith(
      expect.objectContaining({
        apiKey: 'sk-grok',
        baseUrl: 'https://gateway.example',
        model: 'grok-imagine-image',
        prompt: 'draw a cat',
      })
    )
  })
})
