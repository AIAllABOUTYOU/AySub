<template>
  <AppLayout>
    <div class="flex flex-col gap-5 lg:h-[calc(100vh-64px-4rem)]">
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

      <div class="grid gap-5 lg:min-h-0 lg:flex-1 xl:grid-cols-[340px_minmax(0,1fr)]">
        <aside class="card flex flex-col gap-4 p-4 lg:min-h-0">
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

          <div v-if="activeTab === 'video'" class="space-y-4">
            <div>
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.videoSeconds') }}
              </label>
              <select v-model.number="videoSeconds" class="input">
                <option :value="6">6s</option>
                <option :value="10">10s</option>
                <option :value="12">12s</option>
                <option :value="16">16s</option>
                <option :value="20">20s</option>
              </select>
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.videoSize') }}
              </label>
              <select v-model="videoSize" class="input">
                <option value="720x1280">720x1280</option>
                <option value="1280x720">1280x720</option>
                <option value="1024x1024">1024x1024</option>
                <option value="1024x1792">1024x1792</option>
                <option value="1792x1024">1792x1024</option>
              </select>
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.videoResolution') }}
              </label>
              <select v-model="videoResolution" class="input">
                <option value="720p">720p</option>
                <option value="480p">480p</option>
              </select>
            </div>
            <div>
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.videoPreset') }}
              </label>
              <select v-model="videoPreset" class="input">
                <option value="custom">{{ t('experienceCenter.video.presets.custom') }}</option>
                <option value="normal">{{ t('experienceCenter.video.presets.normal') }}</option>
                <option value="fun">{{ t('experienceCenter.video.presets.fun') }}</option>
                <option value="spicy">{{ t('experienceCenter.video.presets.spicy') }}</option>
              </select>
            </div>
          </div>

          <div v-if="activeTab === 'audio'" class="space-y-4">
            <div>
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.audioMode') }}
              </label>
              <div class="grid grid-cols-1 gap-1 rounded-lg border border-gray-200 p-1 sm:grid-cols-3 dark:border-dark-700">
                <button
                  v-for="mode in audioModes"
                  :key="mode"
                  type="button"
                  class="min-h-9 rounded-md px-2 py-2 text-xs font-medium transition-colors"
                  :class="
                    audioMode === mode
                      ? 'bg-primary-500 text-white'
                      : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'
                  "
                  @click="audioMode = mode"
                >
                  {{ t(`experienceCenter.audio.modes.${mode}`) }}
                </button>
              </div>
            </div>
            <div v-if="audioMode === 'speech'">
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.audioVoice') }}
              </label>
              <select v-model="audioVoice" class="input">
                <option v-for="voice in audioVoices" :key="voice" :value="voice">{{ voice }}</option>
              </select>
            </div>
            <div v-if="audioMode === 'speech'">
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.audioFormat') }}
              </label>
              <select v-model="audioFormat" class="input">
                <option value="mp3">mp3</option>
                <option value="opus">opus</option>
                <option value="aac">aac</option>
                <option value="flac">flac</option>
                <option value="wav">wav</option>
                <option value="pcm">pcm</option>
              </select>
            </div>
            <div v-if="audioMode === 'transcription'">
              <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('experienceCenter.controls.audioLanguage') }}
              </label>
              <input v-model="audioLanguage" type="text" class="input" placeholder="zh, en, ja" />
            </div>
          </div>

          <div class="rounded-lg bg-gray-50 p-3 text-xs text-gray-500 dark:bg-dark-700/50 dark:text-gray-400">
            {{ t('experienceCenter.controls.keyHint') }}
          </div>

          <div v-if="activeTab === 'chat'" class="flex min-h-0 flex-1 flex-col border-t border-gray-100 pt-4 dark:border-dark-700">
            <div class="mb-3 flex items-center justify-between gap-2">
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('experienceCenter.sessions.title') }}
              </h2>
              <button type="button" class="btn btn-secondary btn-sm" @click="startNewChat">
                <Icon name="plus" size="sm" />
                {{ t('experienceCenter.sessions.new') }}
              </button>
            </div>
            <div
              v-if="sessionsLoading"
              class="rounded-lg border border-gray-200 p-3 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
            >
              {{ t('common.loading') }}
            </div>
            <div
              v-else-if="chatSessions.length === 0"
              class="rounded-lg border border-dashed border-gray-200 p-3 text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400"
            >
              {{ t('experienceCenter.sessions.empty') }}
            </div>
            <div v-else class="min-h-0 space-y-2 overflow-y-auto pr-1">
              <button
                v-for="session in chatSessions"
                :key="session.id"
                type="button"
                class="group w-full rounded-lg border p-3 text-left transition-colors"
                :class="
                  currentSessionId === session.id
                    ? 'border-primary-300 bg-primary-50 dark:border-primary-800 dark:bg-primary-900/20'
                    : 'border-gray-200 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/50'
                "
                @click="selectSession(session.id)"
              >
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                      {{ session.title || t('experienceCenter.sessions.untitled') }}
                    </div>
                    <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                      {{ session.model || t('experienceCenter.controls.selectModel') }}
                    </div>
                  </div>
                  <button
                    type="button"
                    class="rounded-md p-1 text-gray-400 opacity-100 transition-colors hover:bg-red-50 hover:text-red-600 sm:opacity-0 sm:group-hover:opacity-100 dark:hover:bg-red-900/20 dark:hover:text-red-300"
                    :title="t('experienceCenter.sessions.delete')"
                    :disabled="deletingSessionId === session.id"
                    @click.stop="removeSession(session.id)"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
                <div class="mt-2 flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <span class="truncate">{{ session.api_key_name || t('experienceCenter.sessions.noKey') }}</span>
                  <span>{{ formatSessionTime(session.updated_at) }}</span>
                </div>
              </button>
            </div>
          </div>
        </aside>

        <main class="card overflow-hidden lg:min-h-0">
          <section v-if="activeTab === 'chat'" class="flex min-h-[520px] flex-col lg:h-full lg:min-h-0">
            <div class="flex-1 overflow-y-auto p-4">
              <div v-if="messages.length === 0" class="flex min-h-[280px] items-center justify-center text-sm text-gray-500 dark:text-gray-400 lg:h-full lg:min-h-0">
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
                    class="max-w-[82%] rounded-xl px-3 py-2 text-sm leading-6"
                    :class="
                      message.role === 'user'
                        ? 'whitespace-pre-wrap bg-primary-500 text-white'
                        : 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-gray-100'
                    "
                  >
                    <template v-if="message.role === 'user'">
                      {{ message.content }}
                    </template>
                    <div v-else class="experience-chat-markdown" v-html="renderMarkdown(message.content)"></div>
                  </div>
                </div>

                <!-- AI thinking indicator -->
                <div v-if="submitting" class="flex justify-start">
                  <div class="max-w-[82%] rounded-xl bg-gray-100 px-3 py-2 text-sm leading-6 dark:bg-dark-700">
                    <div class="flex items-center gap-2">
                      <div class="flex gap-1">
                        <span class="ai-thinking-dot"></span>
                        <span class="ai-thinking-dot ai-thinking-dot-delay-1"></span>
                        <span class="ai-thinking-dot ai-thinking-dot-delay-2"></span>
                      </div>
                      <span class="text-gray-600 dark:text-gray-300">{{ t('experienceCenter.chat.thinking') }}</span>
                    </div>
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
                <button type="button" class="btn btn-secondary" @click="clearCurrentChat">
                  {{ currentSessionId ? t('experienceCenter.sessions.new') : t('experienceCenter.actions.clear') }}
                </button>
                <button type="submit" class="btn btn-primary" :disabled="submitting">
                  <Icon name="chat" size="sm" />
                  {{ submitting ? t('experienceCenter.actions.sending') : t('experienceCenter.actions.send') }}
                </button>
              </div>
            </form>
          </section>

          <section v-else-if="activeTab === 'image'" class="flex min-h-[520px] flex-col lg:h-full lg:min-h-0">
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
              <div v-if="generatedImages.length === 0" class="flex min-h-[280px] items-center justify-center text-sm text-gray-500 dark:text-gray-400 lg:h-full lg:min-h-0">
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

          <section v-else-if="activeTab === 'video'" class="flex min-h-[520px] flex-col lg:h-full lg:min-h-0">
            <form class="border-b border-gray-100 p-4 dark:border-dark-700" @submit.prevent="submitVideo">
              <textarea
                v-model="videoPrompt"
                rows="4"
                class="input resize-none"
                :placeholder="t('experienceCenter.video.placeholder')"
              />
              <div class="mt-3 flex justify-end">
                <button type="submit" class="btn btn-primary" :disabled="submitting || videoPolling">
                  <Icon name="play" size="sm" />
                  {{ submitting || videoPolling ? t('experienceCenter.actions.processing') : t('experienceCenter.video.submit') }}
                </button>
              </div>
            </form>

            <div class="flex-1 overflow-y-auto p-4">
              <div v-if="!videoJob" class="flex min-h-[280px] items-center justify-center text-sm text-gray-500 dark:text-gray-400 lg:h-full lg:min-h-0">
                {{ t('experienceCenter.video.empty') }}
              </div>
              <div v-else class="space-y-4">
                <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                  <div class="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div class="font-mono text-xs text-gray-500 dark:text-gray-400">{{ videoJob.id }}</div>
                      <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
                        {{ videoJob.model }}
                      </div>
                    </div>
                    <span
                      class="rounded-full px-3 py-1 text-xs font-medium"
                      :class="videoStatusClass"
                    >
                      {{ videoJob.status }}
                    </span>
                  </div>
                  <div class="mt-4 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                    <div class="h-full bg-primary-500 transition-all" :style="{ width: `${videoProgress}%` }" />
                  </div>
                  <div class="mt-2 flex flex-wrap justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
                    <span>{{ t('experienceCenter.video.progress', { value: videoProgress }) }}</span>
                    <span>{{ videoJob.size }} · {{ videoJob.seconds }}s · {{ videoJob.quality }}</span>
                  </div>
                  <p v-if="videoJob.error" class="mt-3 rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
                    {{ videoErrorMessage }}
                  </p>
                </div>

                <div v-if="videoObjectUrl" class="space-y-3">
                  <video :src="videoObjectUrl" controls class="max-h-[56vh] w-full rounded-lg bg-black" />
                  <div class="flex justify-end">
                    <a class="btn btn-secondary" :href="videoObjectUrl" :download="`aysub-video-${videoJob.id}.mp4`">
                      <Icon name="download" size="sm" />
                      {{ t('experienceCenter.actions.download') }}
                    </a>
                  </div>
                </div>
              </div>
            </div>
          </section>

          <section v-else-if="activeTab === 'audio'" class="flex min-h-[520px] flex-col lg:h-full lg:min-h-0">
            <form class="border-b border-gray-100 p-4 dark:border-dark-700" @submit.prevent="submitAudio">
              <textarea
                v-if="audioMode === 'speech'"
                v-model="audioInput"
                rows="4"
                class="input resize-none"
                :placeholder="t('experienceCenter.audio.speechPlaceholder')"
              />
              <div v-else class="space-y-3">
                <input
                  type="file"
                  accept="audio/*"
                  class="input"
                  @change="handleAudioFileChange"
                />
                <div v-if="audioFile" class="text-xs text-gray-500 dark:text-gray-400">
                  {{ audioFile.name }} · {{ formatBytes(audioFile.size) }}
                </div>
              </div>
              <div class="mt-3 flex justify-end">
                <button type="submit" class="btn btn-primary" :disabled="submitting">
                  <Icon name="bolt" size="sm" />
                  {{ submitting ? t('experienceCenter.actions.processing') : t('experienceCenter.audio.submit') }}
                </button>
              </div>
            </form>

            <div class="flex-1 overflow-y-auto p-4">
              <div v-if="!audioObjectUrl && !audioTextResult" class="flex min-h-[280px] items-center justify-center text-sm text-gray-500 dark:text-gray-400 lg:h-full lg:min-h-0">
                {{ t('experienceCenter.audio.empty') }}
              </div>
              <div v-else class="space-y-4">
                <div v-if="audioObjectUrl" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                  <audio :src="audioObjectUrl" controls class="w-full" />
                  <div class="mt-3 flex justify-end">
                    <a class="btn btn-secondary" :href="audioObjectUrl" :download="`aysub-audio.${audioFormat}`">
                      <Icon name="download" size="sm" />
                      {{ t('experienceCenter.actions.download') }}
                    </a>
                  </div>
                </div>
                <pre
                  v-if="audioTextResult"
                  class="whitespace-pre-wrap rounded-lg border border-gray-200 bg-gray-50 p-4 text-sm leading-6 text-gray-800 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-100"
                >{{ audioTextResult }}</pre>
              </div>
            </div>
          </section>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import userChannelsAPI, { type UserAvailableChannel } from '@/api/channels'
import { BILLING_MODE_IMAGE } from '@/constants/channel'
import {
  appendPlaygroundMessage,
  createPlaygroundSession,
  createChatCompletion,
  createAudioSpeech,
  createAudioTranscription,
  createAudioTranslation,
  createImageGeneration,
  createVideoJob,
  deletePlaygroundSession,
  getPlaygroundSession,
  getVideoContentObjectUrl,
  getVideoJob,
  listPlaygroundSessions,
  resolveGatewayBaseUrl,
  type ChatMessage,
  type GeneratedImage,
  type PlaygroundSession,
  type PlaygroundSessionPayload,
  type VideoJob,
} from '@/api/playground'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { renderMarkdown } from '@/utils/markdown'

type ExperienceTab = 'chat' | 'image' | 'video' | 'audio'
type AudioMode = 'speech' | 'transcription' | 'translation'

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
const sessionsLoading = ref(false)
const deletingSessionId = ref<number | null>(null)
const selectedKeyId = ref(0)
const selectedModel = ref('')
const baseUrl = ref('')
const temperature = ref(0.7)
const imageSize = ref('1024x1024')
const chatInput = ref('')
const imagePrompt = ref('')
const messages = ref<ChatMessage[]>([])
const chatSessions = ref<PlaygroundSession[]>([])
const currentSessionId = ref<number | null>(null)
const generatedImages = ref<GeneratedImage[]>([])
const videoSeconds = ref(6)
const videoSize = ref('720x1280')
const videoResolution = ref('720p')
const videoPreset = ref('custom')
const videoPrompt = ref('')
const videoJob = ref<VideoJob | null>(null)
const videoPolling = ref(false)
const videoObjectUrl = ref('')
const audioModes: AudioMode[] = ['speech', 'transcription', 'translation']
const audioVoices = ['alloy', 'ash', 'ballad', 'coral', 'echo', 'fable', 'nova', 'onyx', 'sage', 'shimmer', 'verse']
const audioMode = ref<AudioMode>('speech')
const audioVoice = ref('alloy')
const audioFormat = ref('mp3')
const audioLanguage = ref('')
const audioInput = ref('')
const audioFile = ref<File | null>(null)
const audioObjectUrl = ref('')
const audioTextResult = ref('')

let videoPollTimer: number | null = null
let videoPollGeneration = 0

const activeKeys = computed(() => apiKeys.value.filter((key) => key.status === 'active' && key.group))
const selectedKey = computed(() => activeKeys.value.find((key) => key.id === selectedKeyId.value) || null)

const availableModels = computed(() => {
  const rows: Array<{ name: string; platform: string; billingMode?: string | null }> = []
  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        rows.push({
          name: model.name,
          platform: model.platform || section.platform,
          billingMode: model.pricing?.billing_mode,
        })
      }
    }
  }
  return rows
})

const selectedModelPlatform = computed(() => {
  const model = availableModels.value.find((m) => m.name === selectedModel.value)
  return model?.platform || ''
})

function modelMatchesActiveTab(model: { name: string; billingMode?: string | null }): boolean {
  const name = model.name.toLowerCase()
  if (activeTab.value === 'image') {
    return (
      model.billingMode === BILLING_MODE_IMAGE ||
      name.startsWith('gpt-image-') ||
      name.startsWith('grok-imagine-image')
    )
  }
  if (activeTab.value === 'video') {
    return name === 'grok-imagine-video' || name.includes('video')
  }
  if (activeTab.value === 'audio') {
    return (
      name.includes('audio') ||
      name.includes('tts') ||
      name.includes('speech') ||
      name.includes('transcrib') ||
      name.includes('whisper')
    )
  }
  return (
    model.billingMode !== BILLING_MODE_IMAGE &&
    !name.startsWith('gpt-image-') &&
    !name.startsWith('grok-imagine-image') &&
    name !== 'grok-imagine-video'
  )
}

const filteredModels = computed(() => {
  const platform = selectedKey.value?.group?.platform
  const names = new Set<string>()
  for (const model of availableModels.value) {
    if ((!platform || model.platform === platform) && modelMatchesActiveTab(model)) {
      names.add(model.name)
    }
  }
  return Array.from(names).sort()
})

const videoProgress = computed(() => {
  const progress = Number(videoJob.value?.progress || 0)
  if (!Number.isFinite(progress)) return 0
  return Math.max(0, Math.min(100, Math.round(progress)))
})

const videoStatusClass = computed(() => {
  const status = videoJob.value?.status
  if (status === 'completed') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (status === 'failed') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
})

const videoErrorMessage = computed(() => {
  const error = videoJob.value?.error
  if (!error) return ''
  const maybeMessage = (error as { message?: unknown }).message
  if (typeof maybeMessage === 'string' && maybeMessage.trim()) return maybeMessage
  return JSON.stringify(error)
})

async function loadData() {
  loading.value = true
  try {
    const [keys, availableChannels, settings, sessions] = await Promise.all([
      keysAPI.list(1, 100, { status: 'active' }),
      userChannelsAPI.getAvailable(),
      appStore.fetchPublicSettings(),
      listPlaygroundSessions(1, 30),
    ])
    apiKeys.value = keys.items
    channels.value = availableChannels
    chatSessions.value = sessions.items.filter((session) => session.mode === 'chat')
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

async function loadSessions() {
  sessionsLoading.value = true
  try {
    const result = await listPlaygroundSessions(1, 30)
    chatSessions.value = result.items.filter((session) => session.mode === 'chat')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    sessionsLoading.value = false
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

function makeChatTitle(prompt: string): string {
  const compact = prompt.replace(/\s+/g, ' ').trim()
  return compact.length > 36 ? `${compact.slice(0, 36)}...` : compact || t('experienceCenter.sessions.untitled')
}

function buildSessionPayload(title: string): PlaygroundSessionPayload {
  return {
    api_key_id: selectedKeyId.value || null,
    title,
    mode: 'chat',
    model: selectedModel.value,
    metadata: {
      temperature: temperature.value,
    },
  }
}

function applySession(session: PlaygroundSession) {
  currentSessionId.value = session.id
  if (session.api_key_id && activeKeys.value.some((key) => key.id === session.api_key_id)) {
    selectedKeyId.value = session.api_key_id
  }
  if (session.model && filteredModels.value.includes(session.model)) {
    selectedModel.value = session.model
  }
  const metadata = session.metadata || {}
  const maybeTemperature = metadata.temperature
  if (typeof maybeTemperature === 'number' && Number.isFinite(maybeTemperature)) {
    temperature.value = Math.max(0, Math.min(2, maybeTemperature))
  }
  messages.value = (session.messages || [])
    .filter((message) => message.role === 'system' || message.role === 'user' || message.role === 'assistant')
    .map((message) => ({ role: message.role, content: message.content }))
}

async function ensureCurrentSession(prompt: string): Promise<number> {
  if (currentSessionId.value) return currentSessionId.value

  const session = await createPlaygroundSession(buildSessionPayload(makeChatTitle(prompt)))
  currentSessionId.value = session.id
  chatSessions.value = [session, ...chatSessions.value.filter((item) => item.id !== session.id)]
  return session.id
}

async function selectSession(sessionID: number) {
  if (submitting.value || currentSessionId.value === sessionID) return
  try {
    const session = await getPlaygroundSession(sessionID)
    applySession(session)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

function startNewChat() {
  currentSessionId.value = null
  messages.value = []
  chatInput.value = ''
}

async function clearCurrentChat() {
  startNewChat()
}

async function removeSession(sessionID: number) {
  deletingSessionId.value = sessionID
  try {
    await deletePlaygroundSession(sessionID)
    chatSessions.value = chatSessions.value.filter((session) => session.id !== sessionID)
    if (currentSessionId.value === sessionID) {
      startNewChat()
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    deletingSessionId.value = null
  }
}

function formatSessionTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
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
    const sessionID = await ensureCurrentSession(prompt)
    await appendPlaygroundMessage(sessionID, userMessage)
    const reply = await createChatCompletion({
      baseUrl: baseUrl.value,
      apiKey: selectedKey.value!.key,
      model: selectedModel.value,
      temperature: temperature.value,
      messages: messages.value,
      platform: selectedModelPlatform.value,
    })
    const assistantMessage: ChatMessage = { role: 'assistant', content: reply }
    messages.value.push(assistantMessage)
    await appendPlaygroundMessage(sessionID, assistantMessage)
    await loadSessions()
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
    clearGeneratedImages()
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

function clearGeneratedImages() {
  for (const image of generatedImages.value) {
    revokeObjectUrl(image.url)
  }
  generatedImages.value = []
}

function revokeObjectUrl(url: string) {
  if (url.startsWith('blob:')) {
    URL.revokeObjectURL(url)
  }
}

function clearVideoPreview() {
  revokeObjectUrl(videoObjectUrl.value)
  videoObjectUrl.value = ''
}

function clearAudioPreview() {
  revokeObjectUrl(audioObjectUrl.value)
  audioObjectUrl.value = ''
  audioTextResult.value = ''
}

function stopVideoPolling() {
  videoPollGeneration++
  videoPolling.value = false
  if (videoPollTimer) {
    window.clearTimeout(videoPollTimer)
    videoPollTimer = null
  }
}

function isVideoTerminal(job: VideoJob | null): boolean {
  return job?.status === 'completed' || job?.status === 'failed'
}

async function loadVideoContent(id: string, apiKey: string, gatewayBaseUrl: string) {
  clearVideoPreview()
  const result = await getVideoContentObjectUrl({
    baseUrl: gatewayBaseUrl,
    apiKey,
    id,
  })
  videoObjectUrl.value = result.url
}

function startVideoPolling(id: string, apiKey: string, gatewayBaseUrl: string) {
  stopVideoPolling()
  videoPolling.value = true
  const generation = videoPollGeneration

  const poll = async () => {
    if (generation !== videoPollGeneration) return
    try {
      const job = await getVideoJob({
        baseUrl: gatewayBaseUrl,
        apiKey,
        id,
      })
      if (generation !== videoPollGeneration) return
      videoJob.value = job
      if (job.status === 'completed') {
        videoPolling.value = false
        await loadVideoContent(job.id, apiKey, gatewayBaseUrl)
        return
      }
      if (isVideoTerminal(job)) {
        videoPolling.value = false
        return
      }
      videoPollTimer = window.setTimeout(poll, 2000)
    } catch (err: unknown) {
      if (generation !== videoPollGeneration) return
      videoPolling.value = false
      appStore.showError(extractApiErrorMessage(err, t('common.error')))
    }
  }

  videoPollTimer = window.setTimeout(poll, 1500)
}

async function submitVideo() {
  const prompt = videoPrompt.value.trim()
  const validationError = validateRunnable()
  if (validationError) {
    appStore.showError(validationError)
    return
  }
  if (!prompt) {
    appStore.showError(t('experienceCenter.errors.enterPrompt'))
    return
  }

  stopVideoPolling()
  clearVideoPreview()
  submitting.value = true
  try {
    const apiKey = selectedKey.value!.key
    const gatewayBaseUrl = baseUrl.value
    const job = await createVideoJob({
      baseUrl: gatewayBaseUrl,
      apiKey,
      model: selectedModel.value,
      prompt,
      seconds: videoSeconds.value,
      size: videoSize.value,
      resolutionName: videoResolution.value,
      preset: videoPreset.value,
    })
    videoJob.value = job
    if (job.status === 'completed') {
      await loadVideoContent(job.id, apiKey, gatewayBaseUrl)
    } else if (!isVideoTerminal(job)) {
      startVideoPolling(job.id, apiKey, gatewayBaseUrl)
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    submitting.value = false
  }
}

function handleAudioFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  audioFile.value = input.files?.[0] || null
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

async function submitAudio() {
  const validationError = validateRunnable()
  if (validationError) {
    appStore.showError(validationError)
    return
  }

  clearAudioPreview()
  submitting.value = true
  try {
    const apiKey = selectedKey.value!.key
    const gatewayBaseUrl = baseUrl.value
    if (audioMode.value === 'speech') {
      const input = audioInput.value.trim()
      if (!input) {
        appStore.showError(t('experienceCenter.errors.enterPrompt'))
        return
      }
      const result = await createAudioSpeech({
        baseUrl: gatewayBaseUrl,
        apiKey,
        model: selectedModel.value,
        input,
        voice: audioVoice.value,
        format: audioFormat.value,
      })
      audioObjectUrl.value = result.url
      return
    }

    if (!audioFile.value) {
      appStore.showError(t('experienceCenter.errors.selectAudioFile'))
      return
    }
    if (audioMode.value === 'transcription') {
      audioTextResult.value = await createAudioTranscription({
        baseUrl: gatewayBaseUrl,
        apiKey,
        model: selectedModel.value,
        file: audioFile.value,
        language: audioLanguage.value.trim() || undefined,
      })
      return
    }
    audioTextResult.value = await createAudioTranslation({
      baseUrl: gatewayBaseUrl,
      apiKey,
      model: selectedModel.value,
      file: audioFile.value,
    })
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    submitting.value = false
  }
}

watch([selectedKeyId, activeTab, filteredModels], syncSelectedModel)
watch(audioMode, clearAudioPreview)

onMounted(loadData)
onBeforeUnmount(() => {
  stopVideoPolling()
  clearGeneratedImages()
  clearVideoPreview()
  clearAudioPreview()
})
</script>

<style scoped>
.experience-chat-markdown {
  overflow-wrap: anywhere;
}

.experience-chat-markdown :deep(p) {
  margin: 0 0 0.5rem;
}

.experience-chat-markdown :deep(p:last-child) {
  margin-bottom: 0;
}

.experience-chat-markdown :deep(ul),
.experience-chat-markdown :deep(ol) {
  margin: 0.5rem 0;
  padding-left: 1.25rem;
}

.experience-chat-markdown :deep(ul) {
  list-style: disc;
}

.experience-chat-markdown :deep(ol) {
  list-style: decimal;
}

.experience-chat-markdown :deep(li) {
  margin: 0.125rem 0;
}

.experience-chat-markdown :deep(a) {
  color: rgb(13 148 136);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.experience-chat-markdown :deep(blockquote) {
  margin: 0.5rem 0;
  border-left: 3px solid rgb(209 213 219);
  padding-left: 0.75rem;
  color: rgb(75 85 99);
}

.dark .experience-chat-markdown :deep(blockquote) {
  border-left-color: rgb(71 85 105);
  color: rgb(203 213 225);
}

.experience-chat-markdown :deep(code) {
  border-radius: 0.25rem;
  background: rgb(229 231 235);
  padding: 0.125rem 0.25rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.875em;
}

.dark .experience-chat-markdown :deep(code) {
  background: rgb(15 23 42);
}

.experience-chat-markdown :deep(pre) {
  margin: 0.5rem 0;
  overflow-x: auto;
  border-radius: 0.5rem;
  background: rgb(17 24 39);
  padding: 0.75rem;
  color: rgb(243 244 246);
}

.experience-chat-markdown :deep(pre code) {
  background: transparent;
  padding: 0;
  color: inherit;
}

.experience-chat-markdown :deep(table) {
  margin: 0.5rem 0;
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.experience-chat-markdown :deep(th),
.experience-chat-markdown :deep(td) {
  border: 1px solid rgb(209 213 219);
  padding: 0.375rem 0.5rem;
  text-align: left;
}

.dark .experience-chat-markdown :deep(th),
.dark .experience-chat-markdown :deep(td) {
  border-color: rgb(71 85 105);
}

/* AI thinking animation */
.ai-thinking-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: rgb(107 114 128);
  animation: ai-thinking-bounce 1.4s infinite ease-in-out both;
}

.dark .ai-thinking-dot {
  background-color: rgb(156 163 175);
}

.ai-thinking-dot-delay-1 {
  animation-delay: -0.32s;
}

.ai-thinking-dot-delay-2 {
  animation-delay: -0.16s;
}

@keyframes ai-thinking-bounce {
  0%, 80%, 100% {
    transform: scale(0.8);
    opacity: 0.5;
  }
  40% {
    transform: scale(1.1);
    opacity: 1;
  }
}
</style>
