<template>
  <div class="relative flex min-h-screen flex-col bg-gray-50 dark:bg-dark-950">
    <header class="relative z-20 px-4 py-4 sm:px-6">
      <nav class="mx-auto flex max-w-6xl items-center justify-between gap-4">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <div class="h-10 w-10 flex-shrink-0 overflow-hidden rounded-xl shadow-md">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="truncate text-lg font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ siteName }}
          </span>
        </router-link>
        <div class="flex flex-shrink-0 items-center gap-2 sm:gap-3">
          <LocaleSwitcher />
          <button
            type="button"
            @click="loadStatus(false)"
            :disabled="loading || refreshing"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 disabled:opacity-50 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="refreshing ? t('publicStatus.refreshing') : t('publicStatus.refresh')"
          >
            <Icon name="refresh" size="md" :class="{ 'animate-spin': refreshing }" />
          </button>
          <button
            type="button"
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
        </div>
      </nav>
    </header>

    <main class="mx-auto w-full max-w-6xl flex-1 px-4 py-8 sm:px-6 sm:py-12">
      <div class="mb-8 flex flex-col gap-5 sm:mb-10 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="mb-2 text-sm font-medium text-primary-600 dark:text-primary-400">
            {{ t('publicStatus.subtitle') }}
          </p>
          <h1 class="text-3xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-4xl">
            {{ t('publicStatus.title') }}
          </h1>
        </div>
        <div
          v-if="status"
          class="inline-flex w-fit items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium"
          :class="statusBadgeClass"
        >
          <span class="h-2.5 w-2.5 rounded-full" :class="statusDotClass"></span>
          {{ statusLabel }}
        </div>
      </div>

      <div v-if="loading" class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <div
          v-for="i in 3"
          :key="i"
          class="rounded-xl border border-gray-200 bg-white p-6 dark:border-dark-700 dark:bg-dark-900"
        >
          <div class="mb-4 h-4 w-24 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
          <div class="h-8 w-32 animate-pulse rounded bg-gray-200 dark:bg-dark-700"></div>
        </div>
      </div>

      <div
        v-else-if="disabled"
        class="rounded-xl border border-gray-200 bg-white p-8 text-center dark:border-dark-700 dark:bg-dark-900"
      >
        <Icon name="lock" size="xl" class="mx-auto mb-4 text-gray-400" />
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('publicStatus.disabledTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
          {{ t('publicStatus.disabledDescription') }}
        </p>
      </div>

      <div
        v-else-if="errorMessage"
        class="rounded-xl border border-red-200 bg-red-50 p-8 text-center dark:border-red-900/60 dark:bg-red-950/30"
      >
        <Icon name="exclamationTriangle" size="xl" class="mx-auto mb-4 text-red-500" />
        <h2 class="text-lg font-semibold text-red-700 dark:text-red-300">
          {{ t('publicStatus.loadFailed') }}
        </h2>
        <p class="mt-2 text-sm text-red-600 dark:text-red-300">
          {{ errorMessage }}
        </p>
      </div>

      <div v-else-if="status" class="space-y-6">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div
            v-for="card in summaryCards"
            :key="card.label"
            class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="mb-3 flex items-center justify-between gap-3">
              <span class="text-sm text-gray-500 dark:text-gray-400">{{ card.label }}</span>
              <Icon :name="card.icon" size="sm" class="text-gray-400" />
            </div>
            <div class="text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">
              {{ card.value }}
            </div>
            <p v-if="card.hint" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              {{ card.hint }}
            </p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <section class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('publicStatus.models') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ modelsSubtitle }}
              </p>
            </div>
            <div class="p-5">
              <div v-if="status.models.visible && modelNames.length > 0" class="flex flex-wrap gap-2">
                <span
                  v-for="model in modelNames"
                  :key="model"
                  class="max-w-full truncate rounded-full border border-gray-200 bg-gray-50 px-3 py-1.5 text-xs font-medium text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100"
                >
                  {{ model }}
                </span>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-gray-400">
                {{ status.models.visible ? t('publicStatus.noModels') : t('publicStatus.hidden') }}
              </p>
            </div>
          </section>

          <section class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('publicStatus.channels') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ channelsSubtitle }}
              </p>
            </div>
            <div class="p-5">
              <div
                v-if="status.channels.visible && channelSummaries.length > 0"
                class="grid grid-cols-1 gap-3 sm:grid-cols-2"
              >
                <div
                  v-for="channel in channelSummaries"
                  :key="channel.platform"
                  class="rounded-lg border border-gray-100 p-4 dark:border-dark-700"
                >
                  <div class="mb-3 font-medium text-gray-900 dark:text-white">
                    {{ channel.platform }}
                  </div>
                  <div class="grid grid-cols-3 gap-3 text-xs">
                    <div>
                      <div class="text-gray-500 dark:text-gray-400">{{ t('publicStatus.active') }}</div>
                      <div class="mt-1 font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                        {{ channel.active }}
                      </div>
                    </div>
                    <div>
                      <div class="text-gray-500 dark:text-gray-400">{{ t('publicStatus.total') }}</div>
                      <div class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">
                        {{ channel.total }}
                      </div>
                    </div>
                    <div>
                      <div class="text-gray-500 dark:text-gray-400">{{ t('publicStatus.modelCountShort') }}</div>
                      <div class="mt-1 font-semibold tabular-nums text-gray-900 dark:text-white">
                        {{ channel.model_count }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <p v-else class="text-sm text-gray-500 dark:text-gray-400">
                {{ status.channels.visible ? t('publicStatus.noChannels') : t('publicStatus.hidden') }}
              </p>
            </div>
          </section>
        </div>

        <section class="rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('publicStatus.recentEvents') }}
            </h2>
          </div>
          <div class="divide-y divide-gray-100 dark:divide-dark-800">
            <div v-if="recentEvents.length === 0" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
              {{ t('publicStatus.noRecentEvents') }}
            </div>
            <div
              v-for="event in recentEvents"
              :key="`${event.created_at}-${event.summary}`"
              class="flex flex-col gap-3 px-5 py-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <span
                    class="rounded-full px-2 py-0.5 text-xs font-medium"
                    :class="eventSeverityClass(event.severity)"
                  >
                    {{ event.severity }}
                  </span>
                  <span class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ event.summary }}
                  </span>
                </div>
                <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
                  <span v-if="event.endpoint">{{ t('publicStatus.endpoint') }}: {{ event.endpoint }}</span>
                  <span v-if="event.status_code">{{ t('publicStatus.statusCode') }}: {{ event.status_code }}</span>
                </div>
              </div>
              <time class="flex-shrink-0 text-xs text-gray-500 dark:text-gray-400">
                {{ formatDateTime(event.created_at) }}
              </time>
            </div>
          </div>
        </section>

        <p class="text-center text-xs text-gray-400 dark:text-dark-500">
          {{ t('publicStatus.generatedAt') }}: {{ formatDateTime(status.generated_at) }}
        </p>
      </div>
    </main>

    <footer class="px-4 py-8 text-center text-sm text-gray-500 dark:text-dark-400 sm:px-6">
      &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { statusAPI, type PublicStatusEvent, type PublicStatusResponse } from '@/api/status'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()

const status = ref<PublicStatusResponse | null>(null)
const loading = ref(true)
const refreshing = ref(false)
const disabled = ref(false)
const errorMessage = ref('')
const isDark = ref(document.documentElement.classList.contains('dark'))

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'AySub')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const currentYear = computed(() => new Date().getFullYear())

const statusLabel = computed(() => {
  if (!status.value) return t('publicStatus.unknown')
  if (status.value.status === 'operational') return t('publicStatus.operational')
  if (status.value.status === 'degraded') return t('publicStatus.degraded')
  return t('publicStatus.unknown')
})

const statusBadgeClass = computed(() => {
  if (status.value?.status === 'operational') {
    return 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-300'
  }
  if (status.value?.status === 'degraded') {
    return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-300'
  }
  return 'border-gray-200 bg-white text-gray-600 dark:border-dark-700 dark:bg-dark-900 dark:text-dark-200'
})

const statusDotClass = computed(() => {
  if (status.value?.status === 'operational') return 'bg-emerald-500'
  if (status.value?.status === 'degraded') return 'bg-amber-500'
  return 'bg-gray-400'
})

const modelNames = computed(() => status.value?.models.names || [])
const channelSummaries = computed(() => status.value?.channels.summaries || [])
const recentEvents = computed(() => status.value?.recent_events || [])

const modelsSubtitle = computed(() => {
  if (!status.value?.models.visible) return t('publicStatus.hidden')
  return t('publicStatus.modelCount', { count: status.value.models.count || modelNames.value.length })
})

const channelsSubtitle = computed(() => {
  if (!status.value?.channels.visible) return t('publicStatus.hidden')
  const active = status.value.channels.active || 0
  const total = status.value.channels.total || 0
  return `${t('publicStatus.activeChannels')}: ${active} / ${total}`
})

const summaryCards = computed(() => {
  const last24h = status.value?.last_24h
  const channels = status.value?.channels
  return [
    {
      label: t('publicStatus.requests'),
      value: formatNumber(last24h?.requests || 0),
      hint: t('publicStatus.last24h'),
      icon: 'chartBar' as const,
    },
    {
      label: t('publicStatus.errorRate'),
      value: formatPercent(last24h?.error_rate || 0),
      hint: t('publicStatus.last24h'),
      icon: 'exclamationCircle' as const,
    },
    {
      label: t('publicStatus.avgLatency'),
      value: formatMs(last24h?.latency_ms?.avg || 0),
      hint: `${formatMs(last24h?.latency_ms?.min_bucket_avg || 0)} - ${formatMs(last24h?.latency_ms?.max_bucket_avg || 0)}`,
      icon: 'clock' as const,
    },
    {
      label: t('publicStatus.activeChannels'),
      value: channels?.visible ? `${channels.active || 0}/${channels.total || 0}` : t('publicStatus.hidden'),
      hint: channels?.visible ? `${t('publicStatus.disabledChannels')}: ${channels.disabled_or_error || 0}` : '',
      icon: 'server' as const,
    },
  ]
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

async function loadStatus(initial: boolean) {
  if (initial) {
    loading.value = true
  } else {
    refreshing.value = true
  }
  disabled.value = false
  errorMessage.value = ''
  try {
    status.value = await statusAPI.getPublicStatus()
  } catch (error: unknown) {
    const err = error as { status?: number; message?: string }
    status.value = null
    if (err.status === 404) {
      disabled.value = true
    } else {
      errorMessage.value = err.message || t('publicStatus.loadFailed')
    }
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat(locale.value).format(value)
}

function formatPercent(value: number): string {
  return `${(value * 100).toFixed(value > 0 && value < 0.01 ? 2 : 1)}%`
}

function formatMs(value: number): string {
  if (!value || value <= 0) return '0ms'
  return `${Math.round(value)}ms`
}

function formatDateTime(value: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(locale.value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function eventSeverityClass(severity: PublicStatusEvent['severity']): string {
  const normalized = String(severity || '').toLowerCase()
  if (normalized === 'critical' || normalized === 'error') {
    return 'bg-red-100 text-red-700 dark:bg-red-950/40 dark:text-red-300'
  }
  if (normalized === 'warning') {
    return 'bg-amber-100 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
  }
  return 'bg-blue-100 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300'
}

onMounted(() => {
  initTheme()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  loadStatus(true)
})
</script>
