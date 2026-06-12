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
  <div v-else class="home-shell min-h-screen overflow-hidden text-slate-100" :class="isDark ? 'home-dark' : 'home-light'">
    <div class="home-grid" aria-hidden="true"></div>

    <!-- Scroll Progress Bar (优化1) -->
    <div class="fixed left-0 top-0 z-50 h-1 bg-gradient-to-r from-cyan-400 to-emerald-400 transition-all duration-300"
         :style="{ width: scrollProgress + '%' }"></div>

    <!-- Loading Skeleton (优化1) -->
    <div v-if="loading" class="fixed inset-0 z-40 flex items-center justify-center bg-[#08090f]">
      <div class="flex flex-col items-center gap-4">
        <div class="h-12 w-12 animate-spin rounded-full border-4 border-cyan-400 border-t-transparent"></div>
        <p class="text-sm text-slate-400">{{ t('common.loading') || '加载中...' }}</p>
      </div>
    </div>

    <header class="sticky top-0 z-30 border-b border-white/10 bg-[#08090f]/88 backdrop-blur-xl"
            :class="{ 'shadow-lg shadow-black/20': scrollY > 50 }">
      <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8" aria-label="主导航">
        <a href="#top" class="flex min-w-0 items-center gap-3" @click="handleHomeLink($event, '#top')" aria-label="返回顶部">
          <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-cyan-300/25 bg-white/8 shadow-lg shadow-cyan-500/10">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-cover" loading="lazy" />
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
            class="relative rounded-md px-3 py-2 text-sm font-medium transition hover:bg-white/8 hover:text-white"
            :class="isActiveNavItem(item.url) ? 'home-nav-active' : 'text-slate-300'"
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
            aria-label="查看文档"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="home-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
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
          <!-- Mobile Menu Button (优化2) -->
          <button
            class="md:hidden rounded-md p-2 text-slate-300 hover:bg-white/8 hover:text-white"
            @click="mobileMenuOpen = !mobileMenuOpen"
            :aria-label="mobileMenuOpen ? '关闭菜单' : '打开菜单'"
            :aria-expanded="mobileMenuOpen"
          >
            <Icon :name="mobileMenuOpen ? 'x' : 'menu'" size="md" />
          </button>
        </div>
      </nav>

      <!-- Mobile Menu (优化2) -->
      <Transition name="slide-down">
        <div v-if="mobileMenuOpen" class="md:hidden border-t border-white/10 bg-[#08090f]/95 backdrop-blur-xl dark:border-white/10 dark:bg-[#08090f]/95">
          <nav class="mx-auto max-w-7xl space-y-1 px-4 py-3" aria-label="移动端导航">
            <a
              v-for="item in visibleNavItems"
              :key="`mobile-${item.label}-${item.url}`"
              :href="item.url"
              class="relative block rounded-md px-3 py-2 text-sm font-medium transition"
              :class="isActiveNavItem(item.url) ? 'home-nav-active' : 'text-slate-300 hover:bg-white/8 hover:text-white'"
              :target="linkTarget(item.url)"
              :rel="linkRel(item.url)"
              @click="handleMobileNavClick($event, item.url)"
            >
              {{ item.label }}
            </a>
            <a
              :href="isAuthenticated ? dashboardPath : '/login'"
              class="block rounded-md border border-cyan-300/30 bg-cyan-300/10 px-3 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16 sm:hidden"
              @click="handleMobileNavClick($event, isAuthenticated ? dashboardPath : '/login')"
            >
              {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
            </a>
          </nav>
        </div>
      </Transition>
    </header>

    <main id="top" class="relative">
      <section class="mx-auto grid min-h-[calc(100vh-64px)] max-w-7xl items-center gap-10 px-4 py-12 sm:px-6 lg:grid-cols-[1fr_500px] lg:px-8 lg:py-16">
        <div class="max-w-3xl">
          <div class="mb-5 inline-flex items-center gap-2 rounded-full border border-emerald-300/25 bg-emerald-300/10 px-4 py-1.5 text-sm font-semibold text-emerald-100">
            <span class="home-pulse h-2 w-2 rounded-full bg-emerald-300"></span>
            {{ resolvedHome.hero_badge }}
          </div>

          <h1 class="max-w-4xl text-4xl font-black leading-tight tracking-normal text-white sm:text-5xl lg:text-6xl">
            {{ resolvedHome.hero_title }}
            <span class="home-shine block">{{ resolvedHome.hero_highlight }}</span>
          </h1>
          <p class="mt-6 max-w-2xl text-base leading-8 text-slate-300 sm:text-lg">
            {{ resolvedHome.hero_description }}
          </p>

          <div class="mt-8 flex flex-col gap-3 sm:flex-row">
            <a
              :href="primaryActionUrl"
              class="inline-flex items-center justify-center rounded-md bg-cyan-300 px-5 py-3 text-sm font-bold text-slate-950 shadow-lg shadow-cyan-500/20 transition hover:-translate-y-0.5 hover:bg-cyan-200"
              @click="handleHomeLink($event, primaryActionUrl)"
            >
              {{ resolvedHome.primary_cta_label }}
              <Icon name="arrowRight" size="sm" class="ml-2" />
            </a>
            <a
              :href="resolvedHome.secondary_cta_url"
              class="inline-flex items-center justify-center rounded-md border border-white/14 bg-white/[0.03] px-5 py-3 text-sm font-semibold text-slate-200 transition hover:border-white/25 hover:bg-white/8"
              :target="linkTarget(resolvedHome.secondary_cta_url)"
              :rel="linkRel(resolvedHome.secondary_cta_url)"
              @click="handleHomeLink($event, resolvedHome.secondary_cta_url)"
            >
              {{ resolvedHome.secondary_cta_label }}
            </a>
          </div>

          <div class="mt-10 grid gap-3 sm:grid-cols-3">
            <div
              v-for="(stat, index) in visibleStats"
              :key="`${stat.value}-${stat.label}`"
              class="animate-on-scroll rounded-lg border border-white/10 bg-white/[0.045] p-4 backdrop-blur"
              :style="{ animationDelay: `${index * 100}ms` }"
            >
              <div class="text-2xl font-semibold text-white">{{ stat.value }}</div>
              <div class="mt-1 text-sm text-slate-400">{{ stat.label }}</div>
            </div>
          </div>
        </div>

        <div class="home-terminal rounded-lg border border-cyan-300/20 bg-[#0d1117] shadow-2xl shadow-cyan-950/35">
          <div class="flex items-center justify-between border-b border-white/10 bg-white/[0.035] px-4 py-3">
            <div class="flex items-center gap-2">
              <span class="h-2.5 w-2.5 rounded-full bg-rose-400"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-amber-300"></span>
              <span class="h-2.5 w-2.5 rounded-full bg-emerald-300"></span>
            </div>
            <div class="text-xs font-medium text-slate-500">{{ resolvedHome.terminal_title }}</div>
          </div>
          <div class="space-y-3 p-5 font-mono text-xs leading-6 text-slate-300 sm:text-sm">
            <div v-for="(line, index) in terminalLines" :key="`${line}-${index}`" class="home-terminal-line">
              <span class="mr-2 select-none" :class="index % 3 === 0 ? 'text-emerald-300' : 'text-cyan-300'"></span>{{ line }}
            </div>
            <div class="home-cursor mt-2 inline-block h-4 w-2 bg-cyan-300/80 align-text-bottom"></div>
          </div>
        </div>
      </section>

      <section id="features" class="border-y border-white/10 bg-white/[0.025] px-4 py-16 sm:px-6 lg:px-8">
        <div class="mx-auto max-w-7xl">
          <div class="max-w-2xl">
            <p class="text-sm font-bold uppercase tracking-normal text-cyan-200">{{ t('home.sections.capabilities') }}</p>
            <h2 class="mt-3 text-3xl font-black text-white">{{ resolvedHome.features_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.features_description }}</p>
          </div>

          <div class="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <article
              v-for="(feature, index) in visibleFeatures"
              :key="feature.title"
              class="animate-on-scroll home-card rounded-lg border border-white/10 bg-white/[0.045] p-5 transition hover:-translate-y-1 hover:border-cyan-300/30 hover:bg-white/[0.07]"
              :style="{ animationDelay: `${index * 100}ms` }"
            >
              <div class="mb-4 flex items-center justify-between gap-3">
                <span class="flex h-11 w-11 items-center justify-center rounded-lg border border-cyan-300/20 bg-cyan-300/10 text-cyan-100">
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
            <p class="text-sm font-bold uppercase tracking-normal text-cyan-200">{{ t('home.sections.models') }}</p>
            <h2 class="mt-3 text-3xl font-black text-white">{{ resolvedHome.models_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.models_description }}</p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2">
            <article
              v-for="model in visibleModels"
              :key="`${model.provider}-${model.name}`"
              class="home-card rounded-lg border border-white/10 bg-white/[0.04] p-4 transition hover:-translate-y-1 hover:border-cyan-300/35"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="flex min-w-0 items-start gap-3">
                  <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg text-sm font-black text-white" :class="modelAccentClass(model.provider)">
                    {{ modelInitial(model) }}
                  </span>
                  <div class="min-w-0">
                    <div class="truncate text-sm font-semibold text-white">{{ model.name }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ model.provider }}</div>
                  </div>
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
            <p class="text-sm font-bold uppercase tracking-normal text-cyan-200">{{ t('home.sections.pricing') }}</p>
            <h2 class="mt-3 text-3xl font-black text-white">{{ resolvedHome.pricing_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.pricing_description }}</p>
          </div>

          <div class="mt-8 grid gap-4 lg:grid-cols-3">
            <article
              v-for="item in visiblePricingItems"
              :key="item.name"
              class="home-card rounded-lg border p-5 transition hover:-translate-y-1"
              :class="item.highlighted ? 'border-cyan-300/40 bg-cyan-300/10 shadow-xl shadow-cyan-950/25' : 'border-white/10 bg-white/[0.045]'"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-semibold text-white">{{ item.name }}</div>
                <span v-if="item.highlighted" class="rounded-full bg-cyan-300 px-2.5 py-1 text-xs font-bold text-slate-950">
                  {{ t('home.recommended') }}
                </span>
              </div>
              <div class="mt-4 flex items-end gap-2">
                <span class="text-4xl font-black text-white">{{ item.price }}</span>
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

      <section
        v-for="(section, sectionIndex) in visibleCustomSections"
        :id="customSectionId(sectionIndex)"
        :key="`${section.title}-${sectionIndex}`"
        class="px-4 py-14 sm:px-6 lg:px-8"
      >
        <div class="mx-auto max-w-7xl">
          <div class="max-w-2xl">
            <p v-if="section.eyebrow" class="text-sm font-bold uppercase tracking-normal text-amber-200">{{ section.eyebrow }}</p>
            <h2 class="mt-3 text-3xl font-black text-white">{{ section.title }}</h2>
            <p v-if="section.description" class="mt-3 text-sm leading-7 text-slate-400">{{ section.description }}</p>
          </div>
          <div
            v-if="section.items?.length"
            class="mt-8 grid gap-4"
            :class="section.layout === 'metrics' ? 'sm:grid-cols-2 lg:grid-cols-4' : 'md:grid-cols-2 lg:grid-cols-3'"
          >
            <article
              v-for="item in visibleSectionItems(section.items)"
              :key="`${item.label}-${item.value}`"
              class="home-card rounded-lg border border-white/10 bg-white/[0.04] p-5"
            >
              <div class="text-xs font-bold uppercase tracking-normal text-slate-500">{{ item.label }}</div>
              <div class="mt-3 text-xl font-bold text-white">{{ item.value }}</div>
              <p v-if="item.description" class="mt-2 text-sm leading-6 text-slate-400">{{ item.description }}</p>
            </article>
          </div>
          <a
            v-if="section.cta_label"
            :href="section.cta_url || primaryActionUrl"
            class="mt-8 inline-flex items-center justify-center rounded-md border border-amber-200/25 bg-amber-200/10 px-5 py-3 text-sm font-semibold text-amber-100 transition hover:bg-amber-200/16"
            :target="linkTarget(section.cta_url || primaryActionUrl)"
            :rel="linkRel(section.cta_url || primaryActionUrl)"
            @click="handleHomeLink($event, section.cta_url || primaryActionUrl)"
          >
            {{ section.cta_label }}
            <Icon name="arrowRight" size="sm" class="ml-2" />
          </a>
        </div>
      </section>

      <section id="info" class="px-4 py-14 sm:px-6 lg:px-8">
        <div class="mx-auto grid max-w-7xl gap-8 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <p class="text-sm font-bold uppercase tracking-normal text-cyan-200">{{ t('home.sections.info') }}</p>
            <h2 class="mt-3 text-3xl font-black text-white">{{ resolvedHome.info_title }}</h2>
            <p class="mt-3 text-sm leading-7 text-slate-400">{{ resolvedHome.info_description }}</p>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <div
              v-for="item in visibleInfoItems"
              :key="`${item.label}-${item.value}`"
              class="rounded-lg border border-white/10 bg-white/[0.04] p-4"
            >
              <div class="text-xs font-bold uppercase tracking-normal text-slate-500">{{ item.label }}</div>
              <div class="mt-2 text-lg font-semibold text-white">{{ item.value }}</div>
              <p v-if="item.description" class="mt-2 text-sm leading-6 text-slate-400">{{ item.description }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="px-4 pb-16 pt-2 sm:px-6 lg:px-8">
        <div class="home-cta mx-auto max-w-7xl overflow-hidden rounded-lg border border-cyan-300/20 bg-cyan-300/10 px-6 py-10 text-center sm:px-10">
          <h2 class="text-3xl font-black text-white">{{ resolvedHome.hero_highlight }}</h2>
          <p class="mx-auto mt-3 max-w-2xl text-sm leading-7 text-cyan-50/75">{{ resolvedHome.hero_description }}</p>
          <a
            :href="primaryActionUrl"
            class="mt-7 inline-flex items-center justify-center rounded-md bg-white px-5 py-3 text-sm font-bold text-cyan-800 transition hover:-translate-y-0.5 hover:bg-cyan-50"
            @click="handleHomeLink($event, primaryActionUrl)"
          >
            {{ resolvedHome.primary_cta_label }}
            <Icon name="arrowRight" size="sm" class="ml-2" />
          </a>
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

    <!-- Back to Top Button (优化2) -->
    <Transition name="fade">
      <button
        v-show="showBackToTop"
        @click="scrollToTop"
        class="fixed bottom-8 right-8 z-40 flex h-12 w-12 items-center justify-center rounded-full bg-cyan-500 text-white shadow-lg transition hover:bg-cyan-400 hover:shadow-xl"
        aria-label="返回顶部"
      >
        <Icon name="arrowUp" size="md" />
      </button>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { isDefaultHomeText } from '@/utils/homeConfigDefaults'
import type {
  HomeConfig,
  HomeCustomSectionItem,
  HomeFeatureItem,
  HomeInfoItem,
  HomeModelItem,
  HomeNavItem,
  HomePricingItem,
  HomeStatItem
} from '@/types'

type HomeIconName = 'server' | 'shield' | 'chart' | 'database' | 'bolt' | 'key' | 'globe' | 'terminal' | 'cloud' | 'cpu' | 'calculator' | 'brain'

interface ResolvedHomeConfig extends Required<Omit<HomeConfig,
  'nav_items' | 'stats' | 'features' | 'models' | 'pricing_items' | 'info_items' | 'terminal_lines' | 'custom_sections'
>> {
  nav_items: HomeNavItem[]
  stats: HomeStatItem[]
  features: HomeFeatureItem[]
  models: HomeModelItem[]
  pricing_items: HomePricingItem[]
  info_items: HomeInfoItem[]
  terminal_lines: string[]
  custom_sections: HomeCustomSectionItem[]
}

const { t, locale } = useI18n()
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

// 新增状态 (优化1-10)
const loading = ref(true)
const scrollY = ref(0)
const scrollProgress = ref(0)
const showBackToTop = ref(false)
const mobileMenuOpen = ref(false)
const activeSection = ref('')

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())

const defaultHomeConfig = computed<ResolvedHomeConfig>(() => {
  locale.value

  return {
    nav_items: [
    { label: t('home.nav.home'), url: '#top', visible: true },
    { label: t('home.nav.features'), url: '#features', visible: true },
    { label: t('home.nav.models'), url: '/models', visible: true },
    { label: t('home.nav.pricing'), url: '#pricing', visible: true },
    { label: t('home.nav.extensions'), url: '#custom-0', visible: false },
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
  ],
  custom_sections: []
  }
})

const resolvedHome = computed<ResolvedHomeConfig>(() => mergeHomeConfig(defaultHomeConfig.value, homeConfig.value))
const visibleNavItems = computed(() => visibleItems(resolvedHome.value.nav_items))
const visibleStats = computed(() => visibleItems(resolvedHome.value.stats))
const visibleFeatures = computed(() => visibleItems(resolvedHome.value.features))
const visibleModels = computed(() => visibleItems(resolvedHome.value.models))
const visiblePricingItems = computed(() => visibleItems(resolvedHome.value.pricing_items))
const visibleInfoItems = computed(() => visibleItems(resolvedHome.value.info_items))
const visibleCustomSections = computed(() => visibleItems(resolvedHome.value.custom_sections))
const terminalLines = computed(() => resolvedHome.value.terminal_lines.filter((line) => line.trim().length > 0))
const primaryActionUrl = computed(() => resolvedHome.value.primary_cta_url || (isAuthenticated.value ? dashboardPath.value : '/login'))

// SEO优化 (优化10) - 使用原生方式
function updateMetaTags() {
  // 更新页面标题
  document.title = `${siteName.value} - ${siteSubtitle.value}`

  // 更新或创建meta标签
  const updateMeta = (name: string, content: string, isProperty = false) => {
    const attribute = isProperty ? 'property' : 'name'
    let meta = document.querySelector(`meta[${attribute}="${name}"]`) as HTMLMetaElement
    if (!meta) {
      meta = document.createElement('meta')
      meta.setAttribute(attribute, name)
      document.head.appendChild(meta)
    }
    meta.content = content
  }

  updateMeta('description', resolvedHome.value.hero_description)
  updateMeta('og:title', `${siteName.value} - ${siteSubtitle.value}`, true)
  updateMeta('og:description', resolvedHome.value.hero_description, true)
  updateMeta('og:image', siteLogo.value || '/logo.png', true)
  updateMeta('og:type', 'website', true)
  updateMeta('twitter:card', 'summary_large_image')
  updateMeta('twitter:title', `${siteName.value} - ${siteSubtitle.value}`)
  updateMeta('twitter:description', resolvedHome.value.hero_description)
}

function mergeHomeConfig(base: ResolvedHomeConfig, custom: HomeConfig | null | undefined): ResolvedHomeConfig {
  const source = custom || {}
  return {
    ...base,
    ...customStrings(base, source),
    nav_items: mergeNavItems(base.nav_items, source.nav_items),
    stats: mergeStats(base.stats, source.stats),
    terminal_lines: mergeStringList(base.terminal_lines, source.terminal_lines),
    features: mergeFeatures(base.features, source.features),
    models: mergeModels(base.models, source.models),
    pricing_items: mergePricingItems(base.pricing_items, source.pricing_items),
    info_items: mergeInfoItems(base.info_items, source.info_items),
    custom_sections: Array.isArray(source.custom_sections) ? source.custom_sections : base.custom_sections
  }
}

function customStrings(base: ResolvedHomeConfig, config: HomeConfig): Partial<ResolvedHomeConfig> {
  const result: Record<string, string> = {}
  for (const [key, value] of Object.entries(config)) {
    if (typeof value === 'string' && value.trim().length > 0) {
      const baseValue = base[key as keyof ResolvedHomeConfig]
      result[key] = resolveText(typeof baseValue === 'string' ? baseValue : '', value)
    }
  }
  return result as Partial<ResolvedHomeConfig>
}

function mergeNavItems(base: HomeNavItem[], source?: HomeNavItem[]): HomeNavItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      label: resolveText(fallback?.label || item.label, item.label)
    }
  })
}

function mergeStats(base: HomeStatItem[], source?: HomeStatItem[]): HomeStatItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      label: resolveText(fallback?.label || item.label, item.label)
    }
  })
}

function mergeStringList(base: string[], source?: string[]): string[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => resolveText(base[index] || item, item))
}

function mergeFeatures(base: HomeFeatureItem[], source?: HomeFeatureItem[]): HomeFeatureItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      title: resolveText(fallback?.title || item.title, item.title),
      description: resolveText(fallback?.description || item.description, item.description),
      tag: item.tag ? resolveText(fallback?.tag || item.tag, item.tag) : item.tag
    }
  })
}

function mergeModels(base: HomeModelItem[], source?: HomeModelItem[]): HomeModelItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      name: item.name || fallback?.name || '',
      provider: item.provider || fallback?.provider || '',
      description: resolveText(fallback?.description || item.description || '', item.description),
      price: resolveText(fallback?.price || item.price || '', item.price),
      status: resolveText(fallback?.status || item.status || '', item.status)
    }
  })
}

function mergePricingItems(base: HomePricingItem[], source?: HomePricingItem[]): HomePricingItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      name: resolveText(fallback?.name || item.name, item.name),
      price: resolveText(fallback?.price || item.price, item.price),
      unit: resolveText(fallback?.unit || item.unit || '', item.unit),
      description: resolveText(fallback?.description || item.description || '', item.description),
      cta_label: resolveText(fallback?.cta_label || item.cta_label || '', item.cta_label),
      features: mergeStringList(fallback?.features || [], item.features)
    }
  })
}

function mergeInfoItems(base: HomeInfoItem[], source?: HomeInfoItem[]): HomeInfoItem[] {
  if (!Array.isArray(source)) return base
  return source.map((item, index) => {
    const fallback = base[index]
    return {
      ...fallback,
      ...item,
      label: resolveText(fallback?.label || item.label, item.label),
      value: resolveText(fallback?.value || item.value, item.value),
      description: resolveText(fallback?.description || item.description || '', item.description)
    }
  })
}

function resolveText(base: string, value?: string): string {
  if (!value || value.trim().length === 0) return base
  return isDefaultHomeText(value) ? base : value
}

function visibleItems<T extends { visible?: boolean }>(items: T[]): T[] {
  return items.filter((item) => item.visible !== false)
}

function visibleSectionItems(items?: HomeInfoItem[]): HomeInfoItem[] {
  return visibleItems(items || [])
}

function resolveIcon(icon?: string): HomeIconName {
  const allowed: HomeIconName[] = ['server', 'shield', 'chart', 'database', 'bolt', 'key', 'globe', 'terminal', 'cloud', 'cpu', 'calculator', 'brain']
  return allowed.includes(icon as HomeIconName) ? icon as HomeIconName : 'server'
}

function modelInitial(model: HomeModelItem): string {
  return (model.name || model.provider || 'AI').trim().slice(0, 1).toUpperCase()
}

function modelAccentClass(provider: string): string {
  const normalized = provider.toLowerCase()
  if (normalized.includes('anthropic') || normalized.includes('claude')) return 'bg-amber-500'
  if (normalized.includes('openai') || normalized.includes('gpt')) return 'bg-emerald-500'
  if (normalized.includes('google') || normalized.includes('gemini')) return 'bg-blue-500'
  if (normalized.includes('xai') || normalized.includes('grok')) return 'bg-fuchsia-500'
  return 'bg-cyan-500'
}

function customSectionId(index: number): string {
  return `custom-${index}`
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

function handleMobileNavClick(event: MouseEvent, url: string) {
  mobileMenuOpen.value = false
  handleHomeLink(event, url)
}

function scrollToTop() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function isActiveNavItem(url: string): boolean {
  if (url.startsWith('#')) {
    const section = url.substring(1)
    // 特殊处理首页/顶部
    if (section === 'top') {
      return activeSection.value === 'top' || activeSection.value === '' || scrollY.value < 100
    }
    return activeSection.value === section
  }
  // 处理 /models 等外部链接
  if (url.startsWith('/')) {
    return window.location.pathname === url
  }
  return false
}

function handleScroll() {
  scrollY.value = window.scrollY
  const windowHeight = window.innerHeight
  const documentHeight = document.documentElement.scrollHeight
  const scrolled = window.scrollY
  const maxScroll = documentHeight - windowHeight
  scrollProgress.value = (scrolled / maxScroll) * 100
  showBackToTop.value = scrollY.value > 300

  // 检测当前激活的section
  const sections = ['top', 'features', 'models', 'pricing', 'info']
  for (const section of sections) {
    const element = document.getElementById(section)
    if (element) {
      const rect = element.getBoundingClientRect()
      if (rect.top <= 100 && rect.bottom >= 100) {
        activeSection.value = section
        break
      }
    }
  }
}

function setupScrollAnimations() {
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('is-visible')
        }
      })
    },
    { threshold: 0.1, rootMargin: '0px 0px -50px 0px' }
  )

  const animatedElements = document.querySelectorAll('.animate-on-scroll')
  animatedElements.forEach((el) => observer.observe(el))
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  const shouldUseDark =
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  isDark.value = shouldUseDark
  document.documentElement.classList.toggle('dark', shouldUseDark)
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }

  // 更新SEO meta标签
  updateMetaTags()

  // 模拟加载完成
  setTimeout(() => {
    loading.value = false
  }, 500)

  // 添加滚动监听
  window.addEventListener('scroll', handleScroll, { passive: true })

  // 设置滚动动画
  setTimeout(() => {
    setupScrollAnimations()
  }, 600)

  // 清理
  return () => {
    window.removeEventListener('scroll', handleScroll)
  }
})

// 监听配置变化，更新SEO标签
watch([siteName, siteSubtitle, resolvedHome], () => {
  updateMetaTags()
}, { deep: true })
</script>

<style scoped>
.home-shell {
  position: relative;
  color: rgb(226 232 240);
  background:
    radial-gradient(circle at 15% 6%, rgba(6, 182, 212, 0.18), transparent 34rem),
    radial-gradient(circle at 82% 30%, rgba(14, 165, 233, 0.1), transparent 30rem),
    linear-gradient(180deg, rgba(8, 10, 18, 1), rgba(2, 6, 23, 1) 46rem, rgba(6, 8, 14, 1));
}

.home-grid {
  pointer-events: none;
  position: fixed;
  inset: 0;
  z-index: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.055) 1px, transparent 1px);
  background-size: 60px 60px;
  mask-image: radial-gradient(ellipse 75% 55% at 50% 22%, black, transparent);
  opacity: 0.55;
}

.home-shell > header {
  position: relative;
  z-index: 30;
}

.home-shell > main,
.home-shell > footer {
  position: relative;
  z-index: 1;
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

.home-shine {
  background: linear-gradient(90deg, #f8fafc 0%, #f8fafc 35%, #67e8f9 50%, #38bdf8 62%, #f8fafc 100%);
  background-clip: text;
  background-size: 200% 100%;
  color: transparent;
  animation: home-shine 4s linear infinite;
  -webkit-background-clip: text;
}

.home-pulse {
  box-shadow: 0 0 0 0 rgba(110, 231, 183, 0.55);
  animation: home-pulse 2s ease-in-out infinite;
}

.home-card {
  box-shadow: 0 16px 50px rgba(0, 0, 0, 0.18);
}

.home-terminal {
  transform: perspective(1200px) rotateX(1deg) rotateY(-2deg);
}

.home-terminal-line {
  overflow-wrap: anywhere;
}

.home-cursor {
  animation: home-cursor 1s step-end infinite;
}

.home-cta {
  background:
    radial-gradient(circle at 25% 35%, rgba(255, 255, 255, 0.16), transparent 35%),
    linear-gradient(135deg, rgba(6, 182, 212, 0.95), rgba(14, 165, 233, 0.8), rgba(56, 189, 248, 0.75));
}

.home-light {
  color: rgb(51 65 85);
  background:
    radial-gradient(circle at 14% 5%, rgba(8, 145, 178, 0.08), transparent 31rem),
    radial-gradient(circle at 86% 28%, rgba(56, 189, 248, 0.06), transparent 30rem),
    linear-gradient(180deg, #ffffff, #f8fafc 46rem, #ffffff);
}

.home-light .home-grid {
  background-image:
    linear-gradient(rgba(15, 23, 42, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 23, 42, 0.04) 1px, transparent 1px);
  opacity: 0.5;
}

.home-light header {
  border-bottom-color: rgba(15, 23, 42, 0.1);
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
}

.home-light h1,
.home-light h2,
.home-light h3,
.home-light .text-white {
  color: rgb(15 23 42);
}

.home-light .text-slate-500 {
  color: rgb(71 85 105);
}

.home-light .text-slate-400 {
  color: rgb(100 116 139);
}

.home-light .text-slate-300,
.home-light .text-slate-200 {
  color: rgb(51 65 85);
}

.home-light .text-cyan-200 {
  color: rgb(8 145 178);
  font-weight: 600;
}

.home-light .text-cyan-100,
.home-light .text-emerald-100,
.home-light .text-amber-100 {
  color: rgb(6 95 120);
  font-weight: 600;
}

.home-light .text-cyan-50\/75 {
  color: rgba(8, 47, 73, 0.85);
}

.home-light .text-slate-950 {
  color: rgb(8 47 73);
}

.home-light .border-white\/10,
.home-light .border-white\/12,
.home-light .border-white\/14 {
  border-color: rgba(15, 23, 42, 0.1);
}

.home-light .border-cyan-300\/20,
.home-light .border-cyan-300\/25,
.home-light .border-cyan-300\/30,
.home-light .border-cyan-300\/35,
.home-light .border-cyan-300\/40 {
  border-color: rgba(8, 145, 178, 0.22);
}

.home-light .bg-white\/8,
.home-light .bg-white\/\[0\.025\],
.home-light .bg-white\/\[0\.03\],
.home-light .bg-white\/\[0\.035\],
.home-light .bg-white\/\[0\.04\],
.home-light .bg-white\/\[0\.045\] {
  background-color: rgba(255, 255, 255, 0.95);
  box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
}

.home-light .bg-cyan-300\/10 {
  background-color: rgba(8, 145, 178, 0.08);
  border: 1px solid rgba(8, 145, 178, 0.15);
}

.home-light .bg-emerald-300\/10 {
  background-color: rgba(16, 185, 129, 0.08);
  border: 1px solid rgba(16, 185, 129, 0.15);
}

.home-light .bg-amber-200\/10 {
  background-color: rgba(217, 119, 6, 0.08);
  border: 1px solid rgba(217, 119, 6, 0.15);
}

.home-light .home-icon-button {
  color: rgb(51 65 85);
  background: transparent;
}

.home-light .home-icon-button:hover {
  background: rgba(15, 23, 42, 0.08);
  color: rgb(15 23 42);
}

.home-light .home-shine {
  background: linear-gradient(90deg, #0f172a 0%, #0f172a 35%, #0891b2 50%, #0284c7 62%, #0f172a 100%);
  background-clip: text;
  background-size: 200% 100%;
  color: transparent;
  -webkit-background-clip: text;
}

.home-light .home-card {
  border: 1px solid rgba(15, 23, 42, 0.1);
  box-shadow: 0 4px 16px rgba(8, 145, 178, 0.08);
}

.home-light .home-card:hover {
  border-color: rgba(8, 145, 178, 0.3);
  box-shadow: 0 8px 24px rgba(8, 145, 178, 0.12);
}

.home-light .home-terminal {
  border-color: rgba(15, 23, 42, 0.15);
  background: #ffffff;
  box-shadow: 0 12px 40px rgba(8, 145, 178, 0.12);
}

.home-light .home-terminal .border-white\/10 {
  border-color: rgba(15, 23, 42, 0.1);
}

.home-light .home-terminal-line {
  color: rgb(51 65 85);
}

.home-light .home-cta {
  background:
    radial-gradient(circle at 25% 35%, rgba(255, 255, 255, 0.5), transparent 35%),
    linear-gradient(135deg, rgba(8, 145, 178, 0.12), rgba(14, 165, 233, 0.1), rgba(56, 189, 248, 0.08));
  border: 1px solid rgba(8, 145, 178, 0.2);
}

/* 日间模式 - CTA区块内文字 */
.home-light .home-cta h2 {
  color: rgb(15 23 42);
}

.home-light .home-cta p {
  color: rgb(51 65 85);
}

.home-light .home-cta a {
  color: rgb(8 47 73);
  background: rgba(8, 145, 178, 0.15);
  border: 1px solid rgba(8, 145, 178, 0.3);
}

.home-light .home-cta a:hover {
  background: rgba(8, 145, 178, 0.25);
}

/* 日间模式 - info区块小标签文字 */
.home-light .text-slate-500,
.home-light .rounded-lg .text-slate-500 {
  color: rgb(100 116 139);
}

/* 日间模式 - info区块描述文字 */
.home-light .text-slate-400,
.home-light .rounded-lg .text-slate-400 {
  color: rgb(71 85 105);
}


/* 新增动画样式 (优化4) */
.animate-on-scroll {
  opacity: 0;
  transform: translateY(30px);
  transition: opacity 0.6s ease-out, transform 0.6s ease-out;
}

.animate-on-scroll.is-visible {
  opacity: 1;
  transform: translateY(0);
}

/* 滑动下拉动画 (优化2) */
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease-out;
}

.slide-down-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.slide-down-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

/* 淡入淡出动画 (优化2) */
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: scale(0.9);
}

/* 响应式优化 (优化7) */
@media (min-width: 1536px) {
  .home-shell > main > section {
    padding-left: 3rem;
    padding-right: 3rem;
  }
}

@media (max-width: 768px) {
  .home-shell > main > section {
    padding-top: 2rem;
    padding-bottom: 2rem;
  }
}

/* 导航选中效果 - 参考模型页面 */
.home-nav-active {
  background: rgba(8, 145, 178, 0.1);
  color: rgb(103 232 249);
}

.home-nav-active::after {
  content: '';
  position: absolute;
  right: 0.7rem;
  bottom: 0.28rem;
  left: 0.7rem;
  height: 2px;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0.8;
}

/* 日间模式导航选中效果 */
.home-light .home-nav-active {
  border: 1px solid rgba(8, 145, 178, 0.18);
  background: rgba(240, 249, 255, 0.9);
  color: rgb(14 116 144);
  box-shadow: 0 6px 18px rgba(8, 145, 178, 0.08);
}

.home-light .home-nav-active::after {
  opacity: 1;
}

/* 日间模式移动端菜单 */
.home-light .md\:hidden.border-t {
  border-color: rgba(15, 23, 42, 0.1);
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
}

.home-light .md\:hidden nav a {
  color: rgb(51 65 85);
}

.home-light .md\:hidden nav a:hover {
  background: rgba(8, 145, 178, 0.08);
  color: rgb(15 23 42);
}


.home-light .home-terminal-line,
.home-light .home-terminal .text-slate-300 {
  color: rgb(51 65 85);
}

.home-light footer {
  border-top-color: rgba(15, 23, 42, 0.08);
}

@keyframes home-shine {
  0% {
    background-position: 200% 0;
  }

  100% {
    background-position: -200% 0;
  }
}

@keyframes home-pulse {
  0%,
  100% {
    box-shadow: 0 0 0 0 rgba(110, 231, 183, 0.45);
  }

  50% {
    box-shadow: 0 0 0 7px rgba(110, 231, 183, 0);
  }
}

@keyframes home-cursor {
  50% {
    opacity: 0;
  }
}

@media (max-width: 1023px) {
  .home-terminal {
    transform: none;
  }
}
</style>
