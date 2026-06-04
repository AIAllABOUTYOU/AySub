<template>
  <BaseDialog :show="show" :title="t('admin.users.userApiKeys')" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div><p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p><p class="text-sm text-gray-500 dark:text-dark-400">{{ user.username }}</p></div>
      </div>
      <div v-if="loading" class="flex justify-center py-8"><svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg></div>
      <div v-else-if="apiKeys.length === 0" class="py-8 text-center"><p class="text-sm text-gray-500">{{ t('admin.users.noApiKeys') }}</p></div>
      <div v-else ref="scrollContainerRef" class="max-h-96 space-y-3 overflow-y-auto" @scroll="closeGroupSelector">
        <div v-for="key in apiKeys" :key="key.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex items-center gap-2"><span class="font-medium text-gray-900 dark:text-white">{{ key.name }}</span><span :class="['badge text-xs', key.status === 'active' ? 'badge-success' : 'badge-danger']">{{ key.status }}</span></div>
              <p class="truncate font-mono text-sm text-gray-500">{{ key.key.substring(0, 20) }}...{{ key.key.substring(key.key.length - 8) }}</p>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-xs"
              :disabled="permissionSaving"
              @click="openPermissionEditor(key)"
            >
              {{ t('admin.users.editKeyPermissions') }}
            </button>
          </div>
          <div class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
            <div class="flex items-center gap-1">
              <span>{{ t('admin.users.group') }}:</span>
              <button
                :ref="(el) => setGroupButtonRef(key.id, el)"
                @click="openGroupSelector(key)"
                class="-mx-1 -my-0.5 flex cursor-pointer items-center gap-1 rounded-md px-1 py-0.5 transition-colors hover:bg-gray-100 dark:hover:bg-dark-700"
                :disabled="updatingKeyIds.has(key.id)"
              >
                <GroupBadge
                  v-if="key.group_id && key.group"
                  :name="key.group.name"
                  :platform="key.group.platform"
                  :subscription-type="key.group.subscription_type"
                  :rate-multiplier="key.group.rate_multiplier"
                />
                <span v-else class="text-gray-400 italic">{{ t('admin.users.none') }}</span>
                <svg v-if="updatingKeyIds.has(key.id)" class="h-3 w-3 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                <svg v-else class="h-3 w-3 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M8.25 15L12 18.75 15.75 15m-7.5-6L12 5.25 15.75 9" /></svg>
              </button>
            </div>
            <div class="flex items-center gap-1"><span>{{ t('admin.users.columns.created') }}: {{ formatDateTime(key.created_at) }}</span></div>
          </div>
          <div class="mt-3 rounded-lg bg-gray-50 px-3 py-2 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">
            <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
              <span class="font-medium text-gray-700 dark:text-dark-100">{{ t('keys.permissions.title') }}</span>
              <span
                :class="[
                  'badge text-xs',
                  key.permission_mode === 'restrict' ? 'badge-warning' : 'badge-info'
                ]"
              >
                {{ key.permission_mode === 'restrict' ? t('keys.permissions.restrict') : t('keys.permissions.inherit') }}
              </span>
              <span v-if="key.permission_mode === 'restrict'">
                {{ t('keys.permissions.modelsCount', { count: key.allowed_models?.length || 0 }) }}
              </span>
              <span v-if="key.permission_mode === 'restrict'">
                {{ t('keys.permissions.endpointsCount', { count: key.allowed_endpoints?.length || 0 }) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>

  <BaseDialog
    :show="permissionDialogVisible"
    :title="t('admin.users.editKeyPermissions')"
    width="wide"
    :z-index="60"
    @close="closePermissionEditor"
  >
    <div v-if="permissionKey" class="space-y-5">
      <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700">
        <p class="font-medium text-gray-900 dark:text-white">{{ permissionKey.name }}</p>
        <p class="mt-1 truncate font-mono text-xs text-gray-500">
          {{ permissionKey.key.substring(0, 20) }}...{{ permissionKey.key.substring(permissionKey.key.length - 8) }}
        </p>
      </div>

      <div>
        <label class="input-label">{{ t('keys.permissions.title') }}</label>
        <Select
          v-model="permissionForm.permission_mode"
          class="w-full sm:max-w-xs"
          :options="permissionModeOptions"
        />
        <p class="input-hint">{{ t('admin.users.keyPermissionsAdminHint') }}</p>
      </div>

      <div v-if="permissionForm.permission_mode === 'restrict'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('keys.permissions.allowedModels') }}</label>
          <textarea
            v-model="permissionForm.allowed_models"
            rows="5"
            class="input font-mono text-sm"
            :placeholder="t('keys.permissions.allowedModelsPlaceholder')"
          />
          <p class="input-hint">{{ t('keys.permissions.allowedModelsHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('keys.permissions.allowedEndpoints') }}</label>
          <div class="grid gap-2 sm:grid-cols-2">
            <label
              v-for="endpoint in permissionEndpointOptions"
              :key="endpoint.value"
              class="flex min-h-[42px] items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 text-sm transition-colors hover:border-primary-300 dark:border-dark-600 dark:hover:border-primary-700"
            >
              <input
                v-model="permissionForm.allowed_endpoints"
                type="checkbox"
                :value="endpoint.value"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-gray-700 dark:text-gray-200">{{ endpoint.label }}</span>
            </label>
          </div>
          <p class="input-hint">{{ t('keys.permissions.allowedEndpointsHint') }}</p>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="closePermissionEditor">
        {{ t('common.cancel') }}
      </button>
      <button
        type="button"
        class="btn btn-primary"
        :disabled="permissionSaving"
        @click="savePermissionEditor"
      >
        {{ permissionSaving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </BaseDialog>

  <!-- Group Selector Dropdown -->
  <Teleport to="body">
    <div
      v-if="groupSelectorKeyId !== null && dropdownPosition"
      ref="dropdownRef"
      class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-[calc(100vw-2rem)] max-w-64 overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 duration-200 dark:bg-dark-800 dark:ring-white/10"
      :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px' }"
    >
      <div class="max-h-64 overflow-y-auto p-1.5">
        <!-- Unbind option -->
        <button
          @click="changeGroup(selectedKeyForGroup!, null)"
          :class="[
            'flex w-full items-center rounded-lg px-3 py-2 text-sm transition-colors',
            !selectedKeyForGroup?.group_id
              ? 'bg-primary-50 dark:bg-primary-900/20'
              : 'hover:bg-gray-100 dark:hover:bg-dark-700'
          ]"
        >
          <span class="text-gray-500 italic">{{ t('admin.users.none') }}</span>
          <svg
            v-if="!selectedKeyForGroup?.group_id"
            class="ml-auto h-4 w-4 shrink-0 text-primary-600 dark:text-primary-400"
            fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"
          ><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" /></svg>
        </button>
        <!-- Group options -->
        <button
          v-for="group in allGroups"
          :key="group.id"
          @click="changeGroup(selectedKeyForGroup!, group.id)"
          :class="[
            'flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors',
            selectedKeyForGroup?.group_id === group.id
              ? 'bg-primary-50 dark:bg-primary-900/20'
              : 'hover:bg-gray-100 dark:hover:bg-dark-700'
          ]"
        >
          <GroupOptionItem
            :name="group.name"
            :platform="group.platform"
            :subscription-type="group.subscription_type"
            :rate-multiplier="group.rate_multiplier"
            :description="group.description"
            :selected="selectedKeyForGroup?.group_id === group.id"
          />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type {
  AdminGroup,
  AdminUser,
  ApiKey,
  ApiKeyPermissionEndpoint,
  ApiKeyPermissionMode
} from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Select from '@/components/common/Select.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])
const { t } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const allGroups = ref<AdminGroup[]>([])
const loading = ref(false)
const updatingKeyIds = ref(new Set<number>())
const groupSelectorKeyId = ref<number | null>(null)
const dropdownPosition = ref<{ top: number; left: number } | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())
const permissionDialogVisible = ref(false)
const permissionSaving = ref(false)
const permissionKey = ref<ApiKey | null>(null)
const permissionForm = ref({
  permission_mode: 'inherit' as ApiKeyPermissionMode,
  allowed_models: '',
  allowed_endpoints: [] as ApiKeyPermissionEndpoint[]
})

const permissionModeOptions = computed(() => [
  { value: 'inherit', label: t('keys.permissions.inherit') },
  { value: 'restrict', label: t('keys.permissions.restrict') }
])

const permissionEndpointOptions = computed<{ value: ApiKeyPermissionEndpoint; label: string }[]>(() => [
  { value: 'chat_completions', label: t('keys.permissions.endpoints.chat_completions') },
  { value: 'responses', label: t('keys.permissions.endpoints.responses') },
  { value: 'messages', label: t('keys.permissions.endpoints.messages') },
  { value: 'embeddings', label: t('keys.permissions.endpoints.embeddings') },
  { value: 'images', label: t('keys.permissions.endpoints.images') },
  { value: 'videos', label: t('keys.permissions.endpoints.videos') },
  { value: 'audio_speech', label: t('keys.permissions.endpoints.audio_speech') },
  { value: 'audio_transcriptions', label: t('keys.permissions.endpoints.audio_transcriptions') },
  { value: 'audio_translations', label: t('keys.permissions.endpoints.audio_translations') },
  { value: 'livekit', label: t('keys.permissions.endpoints.livekit') },
  { value: 'gemini_native', label: t('keys.permissions.endpoints.gemini_native') }
])

const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

watch(() => props.show, (v) => {
  if (v && props.user) {
    load()
    loadGroups()
  } else {
    closeGroupSelector()
    closePermissionEditor()
  }
})

const load = async () => {
  if (!props.user) return
  loading.value = true
  groupButtonRefs.value.clear()
  try {
    const res = await adminAPI.users.getUserApiKeys(props.user.id)
    apiKeys.value = res.items || []
  } catch (error) {
    console.error('Failed to load API keys:', error)
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const groups = await adminAPI.groups.getAll()
    allGroups.value = groups
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const DROPDOWN_HEIGHT = 272 // max-h-64 = 16rem = 256px + padding
const DROPDOWN_GAP = 4

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    closeGroupSelector()
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownWidth = Math.min(256, Math.max(0, window.innerWidth - 32))
      const left = Math.min(Math.max(16, rect.left), Math.max(16, window.innerWidth - dropdownWidth - 16))
      const spaceBelow = window.innerHeight - rect.bottom
      const openUpward = spaceBelow < DROPDOWN_HEIGHT && rect.top > spaceBelow
      dropdownPosition.value = {
        top: openUpward ? rect.top - DROPDOWN_HEIGHT - DROPDOWN_GAP : rect.bottom + DROPDOWN_GAP,
        left
      }
    }
    groupSelectorKeyId.value = key.id
  }
}

const closeGroupSelector = () => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
}

const parseAllowedModels = (text: string): string[] =>
  text
    .split(/\r?\n|,/)
    .map((model) => model.trim())
    .filter(Boolean)

const openPermissionEditor = (key: ApiKey) => {
  closeGroupSelector()
  permissionKey.value = key
  permissionForm.value = {
    permission_mode: key.permission_mode || 'inherit',
    allowed_models: (key.allowed_models || []).join('\n'),
    allowed_endpoints: [...(key.allowed_endpoints || [])]
  }
  permissionDialogVisible.value = true
}

const closePermissionEditor = () => {
  permissionDialogVisible.value = false
  permissionKey.value = null
  permissionForm.value = {
    permission_mode: 'inherit',
    allowed_models: '',
    allowed_endpoints: []
  }
}

const savePermissionEditor = async () => {
  if (!permissionKey.value) return
  const mode = permissionForm.value.permission_mode
  permissionSaving.value = true
  try {
    const result = await adminAPI.apiKeys.updateApiKey(permissionKey.value.id, {
      permission_mode: mode,
      allowed_models: mode === 'restrict' ? parseAllowedModels(permissionForm.value.allowed_models) : [],
      allowed_endpoints: mode === 'restrict' ? [...permissionForm.value.allowed_endpoints] : []
    })
    const idx = apiKeys.value.findIndex((k) => k.id === permissionKey.value?.id)
    if (idx !== -1) {
      apiKeys.value[idx] = result.api_key
    }
    appStore.showSuccess(t('admin.users.keyPermissionsUpdated'))
    closePermissionEditor()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.users.keyPermissionsUpdateFailed'))
  } finally {
    permissionSaving.value = false
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  closeGroupSelector()
  if (key.group_id === newGroupId || (!key.group_id && newGroupId === null)) return

  updatingKeyIds.value.add(key.id)
  try {
    const result = await adminAPI.apiKeys.updateApiKeyGroup(key.id, newGroupId)
    // Update local data
    const idx = apiKeys.value.findIndex((k) => k.id === key.id)
    if (idx !== -1) {
      apiKeys.value[idx] = result.api_key
    }
    if (result.auto_granted_group_access && result.granted_group_name) {
      appStore.showSuccess(t('admin.users.groupChangedWithGrant', { group: result.granted_group_name }))
    } else {
      appStore.showSuccess(t('admin.users.groupChangedSuccess'))
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.users.groupChangeFailed'))
  } finally {
    updatingKeyIds.value.delete(key.id)
  }
}

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && groupSelectorKeyId.value !== null) {
    event.stopPropagation()
    closeGroupSelector()
  }
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (dropdownRef.value && !dropdownRef.value.contains(target)) {
    // Check if the click is on one of the group trigger buttons
    for (const el of groupButtonRefs.value.values()) {
      if (el.contains(target)) return
    }
    closeGroupSelector()
  }
}

const handleClose = () => {
  closeGroupSelector()
  closePermissionEditor()
  emit('close')
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleKeyDown, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeyDown, true)
})
</script>
