<template>
  <Transition name="banner-slide">
    <div
      v-if="visibleAnnouncement"
      class="relative border-b border-blue-200/50 bg-gradient-to-r from-blue-50 via-indigo-50 to-purple-50 dark:border-blue-900/30 dark:from-blue-950/40 dark:via-indigo-950/30 dark:to-purple-950/20"
    >
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-4 px-4 py-2.5 md:px-6">
        <!-- Left: Icon + Content -->
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <!-- Icon -->
          <div class="flex-shrink-0">
            <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 text-white shadow-md">
              <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z" />
              </svg>
            </div>
          </div>

          <!-- Title + Content -->
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="inline-flex items-center rounded-md bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/50 dark:text-blue-300">
                {{ t('announcements.title') }}
              </span>
              <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                {{ visibleAnnouncement.title }}
              </h3>
            </div>
            <p class="mt-0.5 hidden text-xs text-gray-600 dark:text-gray-400 sm:line-clamp-1">
              {{ stripMarkdown(visibleAnnouncement.content) }}
            </p>
          </div>
        </div>

        <!-- Right: Actions -->
        <div class="flex flex-shrink-0 items-center gap-2">
          <!-- View Details Button -->
          <button
            @click="handleViewDetails"
            class="rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white shadow-sm transition-all hover:bg-blue-700 hover:shadow dark:bg-blue-500 dark:hover:bg-blue-600"
          >
            {{ t('common.viewDetails') }}
          </button>

          <!-- Navigation Arrows (if multiple announcements) -->
          <div
            v-if="bannerAnnouncements.length > 1"
            class="flex items-center gap-1 rounded-lg bg-white/50 p-1 backdrop-blur-sm dark:bg-dark-800/50"
          >
            <button
              @click="handlePrevious"
              class="flex h-6 w-6 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-white hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
              :aria-label="t('common.previous')"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <span class="px-1 text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ currentIndex + 1 }}/{{ bannerAnnouncements.length }}
            </span>
            <button
              @click="handleNext"
              class="flex h-6 w-6 items-center justify-center rounded-md text-gray-600 transition-colors hover:bg-white hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white"
              :aria-label="t('common.next')"
            >
              <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
              </svg>
            </button>
          </div>

          <!-- Close Button -->
          <button
            @click="handleDismiss"
            class="flex h-7 w-7 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-white/50 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-dark-800/50 dark:hover:text-gray-300"
            :aria-label="t('common.close')"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { storeToRefs } from 'pinia'
import { useAnnouncementStore } from '@/stores/announcements'

const { t } = useI18n()
const announcementStore = useAnnouncementStore()

const { announcements } = storeToRefs(announcementStore)

// Local state
const currentIndex = ref(0)
const dismissed = ref(false)

// Only show banner-mode announcements that are unread
const bannerAnnouncements = computed(() =>
  announcements.value.filter((a) => a.notify_mode === 'banner' && !a.read_at)
)

const visibleAnnouncement = computed(() => {
  if (dismissed.value || bannerAnnouncements.value.length === 0) {
    return null
  }
  return bannerAnnouncements.value[currentIndex.value]
})

// Strip markdown for preview
function stripMarkdown(text: string): string {
  if (!text) return ''
  return text
    .replace(/[#*_~`>\[\]]/g, '')
    .replace(/\n+/g, ' ')
    .trim()
    .substring(0, 120)
}

function handleViewDetails() {
  if (!visibleAnnouncement.value) return
  // Emit event to open announcement detail modal
  const announcement = visibleAnnouncement.value
  // Mark as read when viewing
  announcementStore.markAsRead(announcement.id)
  // Open the announcement bell modal with this announcement selected
  // We'll need to add this method to the announcement store or emit an event
  window.dispatchEvent(
    new CustomEvent('show-announcement-detail', {
      detail: { announcement },
    })
  )
}

function handlePrevious() {
  if (bannerAnnouncements.value.length <= 1) return
  currentIndex.value =
    (currentIndex.value - 1 + bannerAnnouncements.value.length) %
    bannerAnnouncements.value.length
}

function handleNext() {
  if (bannerAnnouncements.value.length <= 1) return
  currentIndex.value = (currentIndex.value + 1) % bannerAnnouncements.value.length
}

function handleDismiss() {
  dismissed.value = true
  // Optionally mark as read
  if (visibleAnnouncement.value) {
    announcementStore.markAsRead(visibleAnnouncement.value.id)
  }
}

// Auto-rotate through banners every 8 seconds
let rotationTimer: number | null = null

function startRotation() {
  if (bannerAnnouncements.value.length <= 1) return
  rotationTimer = window.setInterval(() => {
    handleNext()
  }, 8000)
}

function stopRotation() {
  if (rotationTimer) {
    clearInterval(rotationTimer)
    rotationTimer = null
  }
}

onMounted(() => {
  // Fetch announcements on mount
  announcementStore.fetchAnnouncements()
  startRotation()
})

onBeforeUnmount(() => {
  stopRotation()
})
</script>

<style scoped>
.banner-slide-enter-active,
.banner-slide-leave-active {
  transition: all 0.3s ease-out;
}

.banner-slide-enter-from {
  transform: translateY(-100%);
  opacity: 0;
}

.banner-slide-leave-to {
  transform: translateY(-100%);
  opacity: 0;
}
</style>
