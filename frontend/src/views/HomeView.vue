<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- SECURITY: homeContent is an admin-only setting. -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Configurable Home Page -->
  <div v-else class="home-shell min-h-screen overflow-hidden text-slate-100">
    <header class="sticky top-0 z-30 border-b border-white/10 bg-slate-950/88 backdrop-blur">
      <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <a href="#top" class="flex min-w-0 items-center gap-3" @click="handleHomeLink($event, '#top')">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-cyan-300/20 bg-white/8">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </span>
          <span class="min-w-0">
            <span class="block truncate text-sm font-semibold text-white">{{ siteName }}</span>
            <span class="hidden truncate text-xs text-slate-400 sm:block">{{ siteSubtitle }}</span>
          </span>
        </a>

        <div class="hidden items-center gap-1 md:flex">
          <a
            v-for="item in visibleNavItems"
            :key="`${item.label}-${item.url}`"
            :href="item.url"
            class="rounded-md px-3 py-2 text-sm font-medium text-slate-300 transition hover:bg-white/8 hover:text-white"
            :target="linkTarget(item.url)"
            :rel="linkRel(item.url)"
            @click="handleHomeLink($event, item.url)"
          >
            {{ item.label }}
          </a>
        </div>

        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="home-icon-button"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="home-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <a
            :href="isAuthenticated ? dashboardPath : '/login'"
            class="hidden rounded-md border border-cyan-300/30 bg-cyan-300/10 px-3 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16 sm:inline-flex"
            @click="handleHomeLink($event, isAuthenticated ? dashboardPath : '/login')"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </a>
        </div>
      </nav>
    </header>

    <main id="top" class="relative">
      <section class="mx-auto grid min-h-[calc(100vh-64px)] max-w-7xl items-center gap-10 px-4 py-12 sm:px-6 lg:grid-cols-[1fr_480px] lg:px-8 lg:py-16">
        <div class="max-w-3xl">
          <div class="mb-5 inline-flex items-center gap-2 rounded-md border border-emerald-300/20 bg-emerald-300/10 px-3 py-1.5 text-sm font-medium text-emerald-100">
            <span class="h-2 w-2 rounded-full bg-emerald-300"></span>
            {{ resolvedHome.hero_badge }}
          </div>

          <h1 class="max-w-4xl text-4xl font-semibold leading-tight tracking-normal text-white sm:text-5xl lg:text-6xl">
            {{ resolvedHome.hero_title }}
            <span class="block text-cyan-200">{{ resolvedHome.hero_highlight }}</span>
          </h1>
          <p class="mt-6 max-w-2xl text-base leading-8 text-slate-300 sm:text-lg">
            {{ resolvedHome.hero_description }}
          </p>

          <div class="mt-8 flex flex-col gap-3 sm:flex-row">
            <a
              :href="primaryActionUrl"
              class="inline-flex items-center justify-center rounded-md bg-cyan-300 px-5 py-3 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200"
              @click="handleHomeLink($event, primaryActionUrl)"
            >
              {{ resolvedHome.primary_cta_label }}
              <Icon name="arrowRight" size="sm" class="ml-2" />
            </a>
            <a
              :href="resolvedHome.secondary_cta_url"
              class="inline-flex items-center justify-center rounded-md border border-white/14 px-5 py-3 text-sm font-semibold text-slate-200 transition hover:border-white/25 hover:bg-white/8"
              :target="linkTarget(resolvedHome.secondary_cta_url)"
              :rel="linkRel(resolvedHome.secondary_cta_url)"
              @click="handleHomeLink($event, resolvedHome.secondary_cta_url)"
            >
              {{ resolvedHome.secondary_cta_label }}
            </a>
          </div>

          <div class="mt-10 grid gap-3 sm:grid-cols-3">
            <div
              v-for="stat in visibleStats"
              :key="`${stat.value}-${stat.label}`"
              class="rounded-lg border border-white/10 bg-white/[0.04] p-4"
            >
              <div class="text-2xl font-semibold text-white">{{ stat.value }}</div>
              <div class="mt-1 text-sm text-slate-400">{{ stat.label }}</div>
            </div>
          </div>
        </div>

        <div class="home-terminal rounded-lg border border-cyan-300/20 bg-slate-950 shadow-2xl shadow-cyan-950/40">
          <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
            <div class="flex items-center gap-2">
              <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-emerald-300"></span>
            </div>
            <div class="text-xs font-medium text-slate-500">{{ resolvedHome.terminal_title }}</div>
          </div>
          <div class="space-y-3 p-4 font-mono text-xs leading-6 text-slate-300 sm:text-sm">
            <div v-for="(line, index) in terminalLines" :key="`${line}-${index}`" class="home-terminal-line">
              <span class="mr-2 select-none text-cyan-300">$</span>{{ line }}
            </div>
          </div>
        </div>
      </section>

      <section id="features" class="border-y border-white/10 bg-white/[0.025] px-4 py-14 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-7xl">
          <div class="max-w-2xl">
            <p class="text-sm font-semibold uppercase tracking-normal text-cyan-200">{{ t('home.sections.capabilities') }}</p>
            <h2 class="mt-3 text-3xl font-semibold text-white">{{ resolvedHome.features_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.features_description }}</p>
          </div>

          <div class="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <article
              v-for="feature in visibleFeatures"
              :key="feature.title"
              class="rounded-lg border border-white/10 bg-slate-900/70 p-5 transition hover:border-cyan-300/30 hover:bg-slate-900"
            >
              <div class="mb-4 flex items-center justify-between gap-3">
                <span class="flex h-10 w-10 items-center justify-center rounded-lg border border-cyan-300/20 bg-cyan-300/10 text-cyan-100">
                  <Icon :name="resolveIcon(feature.icon)" size="md" />
                </span>
                <span v-if="feature.tag" class="rounded-md bg-white/8 px-2 py-1 text-xs font-medium text-slate-300">
                  {{ feature.tag }}
                </span>
              </div>
              <h3 class="text-base font-semibold text-white">{{ feature.title }}</h3>
              <p class="mt-2 text-sm leading-6 text-slate-400">{{ feature.description }}</p>
            </article>
          </div>
        </div>
      </section>

      <section id="models" class="px-4 py-14 sm:px-6 lg:px-8">
        <div class="mx-auto grid max-w-7xl gap-10 lg:grid-cols-[0.8fr_1.2fr]">
          <div>
            <p class="text-sm font-semibold uppercase tracking-normal text-cyan-200">{{ t('home.sections.models') }}</p>
            <h2 class="mt-3 text-3xl font-semibold text-white">{{ resolvedHome.models_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.models_description }}</p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <article
              v-for="model in visibleModels"
              :key="`${model.provider}-${model.name}`"
              class="rounded-lg border border-white/10 bg-white/[0.04] p-4"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold text-white">{{ model.name }}</div>
                  <div class="mt-1 text-xs text-slate-500">{{ model.provider }}</div>
                </div>
                <span class="rounded-md border border-emerald-300/20 bg-emerald-300/10 px-2 py-1 text-xs font-medium text-emerald-100">
                  {{ model.status || t('home.providers.supported') }}
                </span>
              </div>
              <p v-if="model.description" class="mt-3 text-sm leading-6 text-slate-400">{{ model.description }}</p>
              <div v-if="model.price" class="mt-4 text-sm font-semibold text-cyan-200">{{ model.price }}</div>
            </article>
          </div>
        </div>
      </section>

      <section id="pricing" class="border-y border-white/10 bg-white/[0.025] px-4 py-14 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-7xl">
          <div class="max-w-2xl">
            <p class="text-sm font-semibold uppercase tracking-normal text-cyan-200">{{ t('home.sections.pricing') }}</p>
            <h2 class="mt-3 text-3xl font-semibold text-white">{{ resolvedHome.pricing_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.pricing_description }}</p>
          </div>

          <div class="mt-8 grid gap-4 lg:grid-cols-3">
            <article
              v-for="item in visiblePricingItems"
              :key="item.name"
              class="rounded-lg border p-5"
              :class="item.highlighted ? 'border-cyan-300/35 bg-cyan-300/10' : 'border-white/10 bg-slate-900/70'"
            >
              <div class="text-sm font-semibold text-white">{{ item.name }}</div>
              <div class="mt-4 flex items-end gap-2">
                <span class="text-3xl font-semibold text-white">{{ item.price }}</span>
                <span v-if="item.unit" class="pb-1 text-sm text-slate-400">{{ item.unit }}</span>
              </div>
              <p v-if="item.description" class="mt-3 text-sm leading-6 text-slate-400">{{ item.description }}</p>
              <ul class="mt-5 space-y-2">
                <li v-for="feature in item.features || []" :key="feature" class="flex gap-2 text-sm text-slate-300">
                  <Icon name="check" size="sm" class="mt-0.5 shrink-0 text-cyan-200" />
                  <span>{{ feature }}</span>
                </li>
              </ul>
              <a
                v-if="item.cta_label"
                :href="item.cta_url || primaryActionUrl"
                class="mt-6 inline-flex w-full items-center justify-center rounded-md border border-white/12 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-white/8"
                :target="linkTarget(item.cta_url || primaryActionUrl)"
                :rel="linkRel(item.cta_url || primaryActionUrl)"
                @click="handleHomeLink($event, item.cta_url || primaryActionUrl)"
              >
                {{ item.cta_label }}
              </a>
            </article>
          </div>
        </div>
      </section>

      <section id="info" class="px-4 py-14 sm:px-6 lg:px-8">
        <div class="mx-auto grid max-w-7xl gap-8 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <p class="text-sm font-semibold uppercase tracking-normal text-cyan-200">{{ t('home.sections.info') }}</p>
            <h2 class="mt-3 text-3xl font-semibold text-white">{{ resolvedHome.info_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.info_description }}</p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div
              v-for="item in visibleInfoItems"
              :key="`${item.label}-${item.value}`"
              class="rounded-lg border border-white/10 bg-white/[0.04] p-4"
            >
              <div class="text-xs font-medium uppercase tracking-normal text-slate-500">{{ item.label }}</div>
              <div class="mt-2 text-lg font-semibold text-white">{{ item.value }}</div>
              <p v-if="item.description" class="mt-2 text-sm leading-6 text-slate-400">{{ item.description }}</p>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="border-t border-white/10 px-4 py-8 sm:px-6 lg:px-8">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 text-sm text-slate-500 sm:flex-row sm:items-center sm:justify-between">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <div class="flex flex-wrap items-center gap-4">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="hover:text-slate-300">
            {{ t('home.docs') }}
          </a>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer" class="hover:text-slate-300">
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import type {
  HomeConfig,
  HomeFeatureItem,
  HomeInfoItem,
  HomeModelItem,
  HomeNavItem,
  HomePricingItem,
  HomeStatItem
} from '@/types'

type HomeIconName = 'server' | 'shield' | 'chart' | 'database' | 'bolt' | 'key' | 'globe' | 'terminal' | 'cloud' | 'cpu' | 'calculator' | 'brain'

interface ResolvedHomeConfig extends Required<Omit<HomeConfig,
  'nav_items' | 'stats' | 'features' | 'models' | 'pricing_items' | 'info_items' | 'terminal_lines'
>> {
  nav_items: HomeNavItem[]
  stats: HomeStatItem[]
  features: HomeFeatureItem[]
  models: HomeModelItem[]
  pricing_items: HomePricingItem[]
  info_items: HomeInfoItem[]
  terminal_lines: string[]
}

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'AySub')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'All Your AI Sub Hub')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const homeConfig = computed<HomeConfig>(() => appStore.cachedPublicSettings?.home_config || {})

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const githubUrl = 'https://github.com/AIAllABOUTYOU/AySub'

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())

const defaultHomeConfig = computed<ResolvedHomeConfig>(() => ({
  nav_items: [
    { label: t('home.nav.home'), url: '#top', visible: true },
    { label: t('home.nav.features'), url: '#features', visible: true },
    { label: t('home.nav.models'), url: '#models', visible: true },
    { label: t('home.nav.pricing'), url: '#pricing', visible: true },
    { label: t('home.nav.info'), url: '#info', visible: true }
  ],
  hero_badge: t('home.hero.badge'),
  hero_title: t('home.hero.title'),
  hero_highlight: t('home.hero.highlight'),
  hero_description: t('home.hero.description'),
  primary_cta_label: isAuthenticated.value ? t('home.goToDashboard') : t('home.getStarted'),
  primary_cta_url: isAuthenticated.value ? dashboardPath.value : '/login',
  secondary_cta_label: t('home.hero.secondaryCta'),
  secondary_cta_url: '#features',
  stats: [
    { value: '300+', label: t('home.stats.models'), visible: true },
    { value: '99.9%', label: t('home.stats.availability'), visible: true },
    { value: '<200ms', label: t('home.stats.routing'), visible: true }
  ],
  terminal_title: 'aysub-api',
  terminal_lines: [
    'curl https://api.example.com/v1/chat/completions',
    'model: claude-sonnet-4-5',
    'route: group/pro -> healthy channel',
    'usage: logged, billed, audited'
  ],
  features_title: t('home.featuresSection.title'),
  features_description: t('home.featuresSection.description'),
  features: [
    { icon: 'server', title: t('home.features.unifiedGateway'), description: t('home.features.unifiedGatewayDesc'), tag: 'API', visible: true },
    { icon: 'shield', title: t('home.features.multiAccount'), description: t('home.features.multiAccountDesc'), tag: 'Routing', visible: true },
    { icon: 'chart', title: t('home.features.balanceQuota'), description: t('home.features.balanceQuotaDesc'), tag: 'Billing', visible: true },
    { icon: 'key', title: t('home.features.keyAcl'), description: t('home.features.keyAclDesc'), tag: 'ACL', visible: true },
    { icon: 'database', title: t('home.features.logs'), description: t('home.features.logsDesc'), tag: 'Ops', visible: true },
    { icon: 'terminal', title: t('home.features.compatible'), description: t('home.features.compatibleDesc'), tag: 'OpenAI', visible: true }
  ],
  models_title: t('home.modelsSection.title'),
  models_description: t('home.modelsSection.description'),
  models: [
    { name: 'Claude Sonnet / Opus', provider: 'Anthropic', description: t('home.modelsSection.claude'), price: t('home.modelsSection.payAsYouGo'), status: t('home.providers.supported'), visible: true },
    { name: 'GPT-4.1 / GPT-5', provider: 'OpenAI', description: t('home.modelsSection.gpt'), price: t('home.modelsSection.payAsYouGo'), status: t('home.providers.supported'), visible: true },
    { name: 'Gemini 2.5', provider: 'Google', description: t('home.modelsSection.gemini'), price: t('home.modelsSection.payAsYouGo'), status: t('home.providers.supported'), visible: true },
    { name: 'Grok', provider: 'xAI', description: t('home.modelsSection.grok'), price: t('home.modelsSection.payAsYouGo'), status: t('home.providers.supported'), visible: true }
  ],
  pricing_title: t('home.pricingSection.title'),
  pricing_description: t('home.pricingSection.description'),
  pricing_items: [
    {
      name: t('home.pricingSection.starter.name'),
      price: t('home.pricingSection.starter.price'),
      unit: t('home.pricingSection.starter.unit'),
      description: t('home.pricingSection.starter.description'),
      features: [t('home.pricingSection.features.unifiedKey'), t('home.pricingSection.features.usageLogs'), t('home.pricingSection.features.modelSwitch')],
      cta_label: t('home.getStarted'),
      cta_url: '/login',
      visible: true
    },
    {
      name: t('home.pricingSection.team.name'),
      price: t('home.pricingSection.team.price'),
      unit: t('home.pricingSection.team.unit'),
      description: t('home.pricingSection.team.description'),
      features: [t('home.pricingSection.features.quota'), t('home.pricingSection.features.permissions'), t('home.pricingSection.features.reports')],
      cta_label: t('home.getStarted'),
      cta_url: '/login',
      highlighted: true,
      visible: true
    },
    {
      name: t('home.pricingSection.custom.name'),
      price: t('home.pricingSection.custom.price'),
      unit: '',
      description: t('home.pricingSection.custom.description'),
      features: [t('home.pricingSection.features.privateDeploy'), t('home.pricingSection.features.channelPolicy'), t('home.pricingSection.features.audit')],
      cta_label: t('home.pricingSection.custom.cta'),
      cta_url: docUrl.value || '#info',
      visible: true
    }
  ],
  info_title: t('home.infoSection.title'),
  info_description: t('home.infoSection.description'),
  info_items: [
    { label: t('home.infoSection.apiEndpoint'), value: appStore.apiBaseUrl || window.location.origin, description: t('home.infoSection.apiEndpointDesc'), visible: true },
    { label: t('home.infoSection.billing'), value: t('home.infoSection.billingValue'), description: t('home.infoSection.billingDesc'), visible: true },
    { label: t('home.infoSection.security'), value: t('home.infoSection.securityValue'), description: t('home.infoSection.securityDesc'), visible: true },
    { label: t('home.infoSection.contact'), value: appStore.contactInfo || t('home.infoSection.contactValue'), description: t('home.infoSection.contactDesc'), visible: true }
  ]
}))

const resolvedHome = computed<ResolvedHomeConfig>(() => mergeHomeConfig(defaultHomeConfig.value, homeConfig.value))
const visibleNavItems = computed(() => visibleItems(resolvedHome.value.nav_items))
const visibleStats = computed(() => visibleItems(resolvedHome.value.stats))
const visibleFeatures = computed(() => visibleItems(resolvedHome.value.features))
const visibleModels = computed(() => visibleItems(resolvedHome.value.models))
const visiblePricingItems = computed(() => visibleItems(resolvedHome.value.pricing_items))
const visibleInfoItems = computed(() => visibleItems(resolvedHome.value.info_items))
const terminalLines = computed(() => resolvedHome.value.terminal_lines.filter((line) => line.trim().length > 0))
const primaryActionUrl = computed(() => resolvedHome.value.primary_cta_url || (isAuthenticated.value ? dashboardPath.value : '/login'))

function mergeHomeConfig(base: ResolvedHomeConfig, custom: HomeConfig | null | undefined): ResolvedHomeConfig {
  const source = custom || {}
  return {
    ...base,
    ...nonEmptyStrings(source),
    nav_items: Array.isArray(source.nav_items) ? source.nav_items : base.nav_items,
    stats: Array.isArray(source.stats) ? source.stats : base.stats,
    terminal_lines: Array.isArray(source.terminal_lines) ? source.terminal_lines : base.terminal_lines,
    features: Array.isArray(source.features) ? source.features : base.features,
    models: Array.isArray(source.models) ? source.models : base.models,
    pricing_items: Array.isArray(source.pricing_items) ? source.pricing_items : base.pricing_items,
    info_items: Array.isArray(source.info_items) ? source.info_items : base.info_items
  }
}

function nonEmptyStrings(config: HomeConfig): Partial<ResolvedHomeConfig> {
  const result: Record<string, string> = {}
  for (const [key, value] of Object.entries(config)) {
    if (typeof value === 'string' && value.trim().length > 0) {
      result[key] = value
    }
  }
  return result as Partial<ResolvedHomeConfig>
}

function visibleItems<T extends { visible?: boolean }>(items: T[]): T[] {
  return items.filter((item) => item.visible !== false)
}

function resolveIcon(icon?: string): HomeIconName {
  const allowed: HomeIconName[] = ['server', 'shield', 'chart', 'database', 'bolt', 'key', 'globe', 'terminal', 'cloud', 'cpu', 'calculator', 'brain']
  return allowed.includes(icon as HomeIconName) ? icon as HomeIconName : 'server'
}

function isInternalURL(url: string): boolean {
  return url.startsWith('/') || url.startsWith('#')
}

function linkTarget(url: string): string | undefined {
  return url && !isInternalURL(url) ? '_blank' : undefined
}

function linkRel(url: string): string | undefined {
  return url && !isInternalURL(url) ? 'noopener noreferrer' : undefined
}

function handleHomeLink(event: MouseEvent, url: string) {
  if (!url) return
  if (url.startsWith('#')) {
    event.preventDefault()
    const target = document.querySelector(url)
    target?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    return
  }
  if (url.startsWith('/')) {
    event.preventDefault()
    router.push(url)
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.home-shell {
  background:
    linear-gradient(rgba(34, 211, 238, 0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(34, 211, 238, 0.045) 1px, transparent 1px),
    linear-gradient(180deg, rgba(8, 47, 73, 0.38), rgba(2, 6, 23, 0.2) 42rem),
    #020617;
  background-size: 56px 56px, 56px 56px, auto, auto;
}

.home-icon-button {
  display: inline-flex;
  height: 2.25rem;
  width: 2.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.5rem;
  color: rgb(203 213 225);
  transition: background-color 160ms ease, color 160ms ease;
}

.home-icon-button:hover {
  background: rgba(255, 255, 255, 0.08);
  color: white;
}

.home-terminal {
  transform: perspective(1200px) rotateX(1deg) rotateY(-2deg);
}

.home-terminal-line {
  overflow-wrap: anywhere;
}

@media (max-width: 1023px) {
  .home-terminal {
    transform: none;
  }
}
</style>
