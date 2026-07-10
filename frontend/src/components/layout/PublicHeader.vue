<template>
  <header
    class="sticky top-0 z-30 border-b border-white/10 bg-[#08090f]/88 backdrop-blur-xl"
    :class="{ 'shadow-lg shadow-black/20': scrollY > 50 }"
  >
    <nav class="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8" aria-label="主导航">
      <!-- Logo and Site Name -->
      <a :href="homeUrl" class="flex min-w-0 items-center gap-3" @click="handleNavClick($event, homeUrl)" aria-label="返回首页">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg border border-cyan-300/25 bg-white/8 shadow-lg shadow-cyan-500/10">
          <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-cover" loading="lazy" />
        </span>
        <span class="min-w-0">
          <span class="block truncate text-sm font-semibold text-white">{{ siteName }}</span>
          <span class="hidden truncate text-xs text-slate-400 sm:block">{{ siteSubtitle }}</span>
        </span>
      </a>

      <!-- Desktop Navigation -->
      <div class="hidden items-center gap-1 md:flex">
        <a
          v-for="item in visibleNavItems"
          :key="`${item.label}-${item.url}`"
          :href="item.url"
          class="relative rounded-md px-3 py-2 text-sm font-medium transition hover:bg-white/10 hover:text-white"
          :class="isActiveNavItem(item.url) ? 'home-nav-active' : 'text-slate-300'"
          :target="linkTarget(item.url)"
          :rel="linkRel(item.url)"
          @click="handleNavClick($event, item.url)"
        >
          {{ item.label }}
        </a>
      </div>

      <!-- Right Side Actions -->
      <div class="relative z-50 flex items-center gap-2">
        <AnnouncementBell />
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
          :href="isAuthenticated ? dashboardUrl : '/login'"
          class="hidden rounded-md border border-cyan-300/30 bg-cyan-300/10 px-3 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16 sm:inline-flex"
          @click="handleNavClick($event, isAuthenticated ? dashboardUrl : '/login')"
        >
          {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
        </a>
        <!-- Mobile Menu Button -->
        <button
          class="md:hidden rounded-md p-2 text-slate-300 hover:bg-white/10 hover:text-white"
          @click="mobileMenuOpen = !mobileMenuOpen"
          :aria-label="mobileMenuOpen ? '关闭菜单' : '打开菜单'"
          :aria-expanded="mobileMenuOpen"
        >
          <Icon :name="mobileMenuOpen ? 'x' : 'menu'" size="md" />
        </button>
      </div>
    </nav>

    <!-- Mobile Menu -->
    <Transition name="slide-down">
      <div v-if="mobileMenuOpen" class="md:hidden border-t border-white/10 bg-[#08090f]/95 backdrop-blur-xl">
        <nav class="mx-auto max-w-7xl space-y-1 px-4 py-3" aria-label="移动端导航">
          <a
            v-for="item in visibleNavItems"
            :key="`mobile-${item.label}-${item.url}`"
            :href="item.url"
            class="relative block rounded-md px-3 py-2 text-sm font-medium transition"
            :class="isActiveNavItem(item.url) ? 'home-nav-active' : 'text-slate-300 hover:bg-white/10 hover:text-white'"
            :target="linkTarget(item.url)"
            :rel="linkRel(item.url)"
            @click="handleMobileNavClick($event, item.url)"
          >
            {{ item.label }}
          </a>
          <a
            :href="isAuthenticated ? dashboardUrl : '/login'"
            class="block rounded-md border border-cyan-300/30 bg-cyan-300/10 px-3 py-2 text-sm font-semibold text-cyan-100 transition hover:bg-cyan-300/16 sm:hidden"
            @click="handleMobileNavClick($event, isAuthenticated ? dashboardUrl : '/login')"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </a>
        </nav>
      </div>
    </Transition>
  </header>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'
import type { HomeNavItem } from '@/types'
import { sanitizeUrl } from '@/utils/url'

interface Props {
  currentPath?: string
  homeUrl?: string
  dashboardUrl?: string
}

const props = withDefaults(defineProps<Props>(), {
  currentPath: '/',
  homeUrl: '/home',
  dashboardUrl: '/console'
})

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const mobileMenuOpen = ref(false)
const scrollY = ref(0)
const isDark = ref(false)

// Computed
const isAuthenticated = computed(() => authStore.isAuthenticated)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'AySub-演示站')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'All Your AI Sub Hub')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))

// Default navigation items
const defaultNavItems = computed<HomeNavItem[]>(() => [
  { label: t('home.nav.home'), url: '/home', visible: true },
  { label: t('home.nav.features'), url: '#features', visible: true },
  { label: t('home.nav.models'), url: '/models', visible: true },
  { label: t('home.nav.pricing'), url: '#pricing', visible: true },
  { label: t('home.nav.info'), url: '#info', visible: true },
])

// Get navigation items from config or use defaults
const visibleNavItems = computed(() => {
  const configured = appStore.cachedPublicSettings?.home_config?.nav_items
  // Check if configured items have valid labels, otherwise use defaults
  const hasValidLabels = Array.isArray(configured) && configured.some(item => item.label && item.label.trim())
  return (hasValidLabels ? configured : defaultNavItems.value)
    .filter((item) => item.visible !== false)
})

// Helper functions
function linkTarget(url: string) {
  return url.startsWith('http') ? '_blank' : '_self'
}

function linkRel(url: string) {
  return url.startsWith('http') ? 'noopener noreferrer' : undefined
}

function isActiveNavItem(url: string) {
  if (url.startsWith('#')) return false
  if (url.startsWith('http')) return false
  return props.currentPath === url || props.currentPath.startsWith(url + '/')
}

function handleNavClick(event: MouseEvent, url: string) {
  // Handle internal navigation
  if (!url.startsWith('http') && !url.startsWith('#')) {
    event.preventDefault()
    router.push(url)
  } else if (url.startsWith('#')) {
    // Smooth scroll for anchor links
    event.preventDefault()
    const element = document.querySelector(url)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth' })
    }
  }
}

function handleMobileNavClick(event: MouseEvent, url: string) {
  mobileMenuOpen.value = false
  handleNavClick(event, url)
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function handleScroll() {
  scrollY.value = window.scrollY
}

// Lifecycle
onMounted(() => {
  // Initialize theme
  const savedTheme = localStorage.getItem('theme')
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  isDark.value = savedTheme === 'dark' || (!savedTheme && prefersDark)
  document.documentElement.classList.toggle('dark', isDark.value)

  // Scroll listener
  window.addEventListener('scroll', handleScroll, { passive: true })

  // Load public settings if not already loaded
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style scoped>
.home-icon-button {
  @apply flex h-9 w-9 items-center justify-center rounded-lg text-slate-300 transition-all hover:bg-white/10 hover:text-white hover:scale-105;
}

.home-nav-active {
  @apply text-white bg-white/10;
}

.home-nav-active::after {
  content: '';
  @apply absolute bottom-0 left-1/2 h-0.5 w-4 -translate-x-1/2 rounded-full bg-cyan-300;
}

/* Mobile menu animation */
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.3s ease-out;
}

.slide-down-enter-from {
  transform: translateY(-100%);
  opacity: 0;
}

.slide-down-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}
</style>
