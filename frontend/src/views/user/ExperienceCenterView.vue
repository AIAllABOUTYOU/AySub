<template>
  <AppLayout>
    <div class="flex h-[calc(100vh-64px-4rem)] flex-col gap-5">
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]">
        <div class="flex flex-wrap items-center gap-2 rounded-xl border border-gray-200 bg-white p-2 dark:border-dark-700 dark:bg-dark-800">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            class="inline-flex h-10 items-center gap-2 rounded-lg px-3 text-sm font-medium transition-colors"
            :class="
              activeTab === tab.value
                ? 'bg-primary-500 text-white shadow-sm'
                : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'
            "
            @click="activeTab = tab.value"
          >
            <Icon :name="tab.icon" size="sm" />
            {{ tab.label }}
          </button>
        </div>

        <button
          @click="loadData"
          :disabled="loading"
          class="btn btn-secondary justify-center"
          :title="t('common.refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          {{ t('common.refresh') }}
        </button>
      </div>

      <div class="grid min-h-0 flex-1 gap-5 xl:grid-cols-[340px_minmax(0,1fr)]">
        <aside class="card flex min-h-0 flex-col gap-4 p-4">
          <div>
            <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('experienceCenter.controls.apiKey') }}
            </label>
            <select v-model="selectedKeyId" class="input">
              <option :value="0">{{ t('experienceCenter.controls.selectKey') }}</option>
              <option v-for="key in activeKeys" :key="key.id" :value="key.id">
                {{ key.name }} · {{ key.group?.name || t('keys.noGroup') }}
              </option>
            </select>
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('experienceCenter.controls.model') }}
            </label>
            <select v-model="selectedModel" class="input">
              <option value="">{{ t('experienceCenter.controls.selectModel') }}</option>
              <option v-for="model in filteredModels" :key="model" :value="model">
                {{ model }}
              </option>
            </select>
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('experienceCenter.controls.baseUrl') }}
            </label>
            <input v-model="baseUrl" type="text" class="input font-mono text-xs" />
          </div>

          <div v-if="activeTab === 'chat'">
            <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('experienceCenter.controls.temperature') }}
            </label>
            <input v-model.number="temperature" type="range" min="0" max="2" step="0.1" class="w-full" />
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ temperature.toFixed(1) }}</div>
          </div>

          <div v-if="activeTab === 'image'">
            <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('experienceCenter.controls.imageSize') }}
            </label>
            <select v-model="imageSize" class="input">
              <option value="1024x1024">1024x1024</option>
              <option value="1024x1536">1024x1536</option>
              <option value="1536x1024">1536x1024</option>
              <option value="1792x1024">1792x1024</option>
              <option value="1024x1792">1024x1792</option>
            </select>
          </div>

          <div class="rounded-lg bg-gray-50 p-3 text-xs text-gray-500 dark:bg-dark-700/50 dark:text-gray-400">
            {{ t('experienceCenter.controls.keyHint') }}
          </div>
        </aside>

        <main class="card min-h-0 overflow-hidden">
          <section v-if="activeTab === 'chat'" class="flex h-full min-h-0 flex-col">
            <div class="flex-1 overflow-y-auto p-4">
              <div v-if="messages.length === 0" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('experienceCenter.chat.empty') }}
              </div>
              <div v-else class="space-y-3">
                <div
                  v-for="(message, index) in messages"
                  :key="index"
                  class="flex"
                  :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
                >
                  <div
                    class="max-w-[82%] whitespace-pre-wrap rounded-xl px-3 py-2 text-sm leading-6"
                    :class="
                      message.role === 'user'
                        ? 'bg-primary-500 text-white'
                        : 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-gray-100'
                    "
                  >
                    {{ message.content }}
                  </div>
                </div>
              </div>
            </div>

            <form class="border-t border-gray-100 p-4 dark:border-dark-700" @submit.prevent="sendChat">
              <textarea
                v-model="chatInput"
                rows="4"
                class="input resize-none"
                :placeholder="t('experienceCenter.chat.placeholder')"
              />
              <div class="mt-3 flex justify-end gap-2">
                <button type="button" class="btn btn-secondary" @click="messages = []">
                  {{ t('experienceCenter.actions.clear') }}
                </button>
                <button type="submit" class="btn btn-primary" :disabled="submitting">
                  <Icon name="chat" size="sm" />
                  {{ submitting ? t('experienceCenter.actions.sending') : t('experienceCenter.actions.send') }}
                </button>
              </div>
            </form>
          </section>

          <section v-else-if="activeTab === 'image'" class="flex h-full min-h-0 flex-col">
            <form class="border-b border-gray-100 p-4 dark:border-dark-700" @submit.prevent="generateImage">
              <textarea
                v-model="imagePrompt"
                rows="4"
                class="input resize-none"
                :placeholder="t('experienceCenter.image.placeholder')"
              />
              <div class="mt-3 flex justify-end">
                <button type="submit" class="btn btn-primary" :disabled="submitting">
                  <Icon name="sparkles" size="sm" />
                  {{ submitting ? t('experienceCenter.actions.generating') : t('experienceCenter.actions.generate') }}
                </button>
              </div>
            </form>

            <div class="flex-1 overflow-y-auto p-4">
              <div v-if="generatedImages.length === 0" class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('experienceCenter.image.empty') }}
              </div>
              <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                <figure
                  v-for="image in generatedImages"
                  :key="image.url"
                  class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
                >
                  <img :src="image.url" alt="" class="aspect-square w-full object-cover" />
                  <figcaption v-if="image.revisedPrompt" class="p-3 text-xs text-gray-500 dark:text-gray-400">
                    {{ image.revisedPrompt }}
                  </figcaption>
                </figure>
              </div>
            </div>
          </section>

          <section v-else class="flex h-full items-center justify-center p-8">
            <div class="max-w-sm text-center">
              <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-300">
                <Icon :name="activeTab === 'video' ? 'play' : 'bolt'" size="lg" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t(`experienceCenter.${activeTab}.title`) }}
              </h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ t('experienceCenter.comingSoon') }}
              </p>
            </div>
          </section>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import {
  createChatCompletion,
  createImageGeneration,
  resolveGatewayBaseUrl,
  type ChatMessage,
  type GeneratedImage,
} from '@/api/playground'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'

type ExperienceTab = 'chat' | 'image' | 'video' | 'audio'

const { t } = useI18n()
const appStore = useAppStore()

const tabs = computed(() => [
  { value: 'chat' as const, label: t('experienceCenter.tabs.chat'), icon: 'chat' as const },
  { value: 'image' as const, label: t('experienceCenter.tabs.image'), icon: 'sparkles' as const },
  { value: 'video' as const, label: t('experienceCenter.tabs.video'), icon: 'play' as const },
  { value: 'audio' as const, label: t('experienceCenter.tabs.audio'), icon: 'bolt' as const },
])

const activeTab = ref<ExperienceTab>('chat')
const apiKeys = ref<ApiKey[]>([])
const channels = ref<UserAvailableChannel[]>([])
const loading = ref(false)
const submitting = ref(false)
const selectedKeyId = ref(0)
const selectedModel = ref('')
const baseUrl = ref('')
const temperature = ref(0.7)
const imageSize = ref('1024x1024')
const chatInput = ref('')
const imagePrompt = ref('')
const messages = ref<ChatMessage[]>([])
const generatedImages = ref<GeneratedImage[]>([])

const activeKeys = computed(() => apiKeys.value.filter((key) => key.status === 'active' && key.group))
const selectedKey = computed(() => activeKeys.value.find((key) => key.id === selectedKeyId.value) || null)

const availableModels = computed(() => {
  const rows: Array<{ name: string; platform: string }> = []
  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        rows.push({ name: model.name, platform: model.platform || section.platform })
      }
    }
  }
  return rows
})

const filteredModels = computed(() => {
  const platform = selectedKey.value?.group?.platform
  const names = new Set<string>()
  for (const model of availableModels.value) {
    if (!platform || model.platform === platform) {
      names.add(model.name)
    }
  }
  return Array.from(names).sort()
})

async function loadData() {
  loading.value = true
  try {
    const [keys, availableChannels, settings] = await Promise.all([
      keysAPI.list(1, 100, { status: 'active' }),
      userChannelsAPI.getAvailable(),
      appStore.fetchPublicSettings(),
    ])
    apiKeys.value = keys.items
    channels.value = availableChannels
    baseUrl.value = resolveGatewayBaseUrl(settings?.api_base_url || appStore.apiBaseUrl)
    if (!selectedKeyId.value && activeKeys.value.length > 0) {
      selectedKeyId.value = activeKeys.value[0].id
    }
    syncSelectedModel()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

function syncSelectedModel() {
  if (!filteredModels.value.includes(selectedModel.value)) {
    selectedModel.value = filteredModels.value[0] || ''
  }
}

function validateRunnable(): string | null {
  if (!selectedKey.value) return t('experienceCenter.errors.selectKey')
  if (!selectedModel.value) return t('experienceCenter.errors.selectModel')
  return null
}

async function sendChat() {
  const prompt = chatInput.value.trim()
  const validationError = validateRunnable()
  if (validationError) {
    appStore.showError(validationError)
    return
  }
  if (!prompt) {
    appStore.showError(t('experienceCenter.errors.enterPrompt'))
    return
  }

  const userMessage: ChatMessage = { role: 'user', content: prompt }
  messages.value.push(userMessage)
  chatInput.value = ''
  submitting.value = true
  try {
    const reply = await createChatCompletion({
      baseUrl: baseUrl.value,
      apiKey: selectedKey.value!.key,
      model: selectedModel.value,
      temperature: temperature.value,
      messages: messages.value,
    })
    messages.value.push({ role: 'assistant', content: reply })
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    submitting.value = false
  }
}

async function generateImage() {
  const prompt = imagePrompt.value.trim()
  const validationError = validateRunnable()
  if (validationError) {
    appStore.showError(validationError)
    return
  }
  if (!prompt) {
    appStore.showError(t('experienceCenter.errors.enterPrompt'))
    return
  }

  submitting.value = true
  try {
    generatedImages.value = await createImageGeneration({
      baseUrl: baseUrl.value,
      apiKey: selectedKey.value!.key,
      model: selectedModel.value,
      prompt,
      size: imageSize.value,
    })
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    submitting.value = false
  }
}

watch([selectedKeyId, filteredModels], syncSelectedModel)

onMounted(loadData)
</script>
