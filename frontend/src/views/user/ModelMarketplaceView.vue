<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="flex flex-col justify-between gap-4 xl:flex-row xl:items-start">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <div class="relative w-full sm:w-80">
                <Icon
                  name="search"
                  size="md"
                  class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
                />
                <input
                  v-model="searchQuery"
                  type="text"
                  :placeholder="t('modelMarketplace.searchPlaceholder')"
                  class="input pl-10"
                />
              </div>

              <select v-model="selectedPlatform" class="input w-full sm:w-52">
                <option value="">{{ t('modelMarketplace.filters.allPlatforms') }}</option>
                <option v-for="platform in platformOptions" :key="platform" :value="platform">
                  {{ platformLabel(platform) }}
                </option>
              </select>
            </div>

            <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-between gap-3 xl:w-auto xl:justify-end">
              <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">
                  {{ t('modelMarketplace.stats.models', { count: modelRows.length }) }}
                </span>
                <span class="rounded-md bg-gray-100 px-2 py-1 dark:bg-dark-700">
                  {{ t('modelMarketplace.stats.channels', { count: channelCount }) }}
                </span>
              </div>

              <button
                @click="loadChannels"
                :disabled="loading"
                class="btn btn-secondary"
                :title="t('common.refresh', 'Refresh')"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
            </div>
          </div>

          <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-3 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('modelMarketplace.calculator.title') }}
              </h3>
              <button type="button" class="btn btn-secondary btn-sm" @click="resetCalculatorInputs">
                {{ t('modelMarketplace.calculator.reset') }}
              </button>
            </div>

            <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.model') }}</span>
                <select v-model="calculatorForm.modelKey" class="input">
                  <option value="">{{ t('modelMarketplace.calculator.selectModel') }}</option>
                  <option v-for="option in calculatorModelOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.group') }}</span>
                <select v-model.number="calculatorForm.groupId" class="input">
                  <option :value="0">{{ t('modelMarketplace.calculator.selectGroup') }}</option>
                  <option v-for="group in calculatorGroups" :key="group.id" :value="group.id">
                    {{ group.name }} · {{ calculatorGroupMultiplier(group).toFixed(4) }}x
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.channel') }}</span>
                <select v-model="calculatorForm.channelName" class="input">
                  <option value="">{{ t('modelMarketplace.calculator.selectChannel') }}</option>
                  <option v-for="channelName in calculatorChannelOptions" :key="channelName" :value="channelName">
                    {{ channelName }}
                  </option>
                </select>
              </label>

              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.billingMode') }}</span>
                <select v-model="calculatorForm.billingMode" class="input">
                  <option v-for="option in calculatorBillingModeOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </label>
            </div>

            <div
              v-if="calculatorForm.billingMode === BILLING_MODE_TOKEN"
              class="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.inputTokens') }}</span>
                <input v-model.number="calculatorForm.inputTokens" min="0" type="number" class="input" />
              </label>
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.outputTokens') }}</span>
                <input v-model.number="calculatorForm.outputTokens" min="0" type="number" class="input" />
              </label>
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.cacheWriteTokens') }}</span>
                <input v-model.number="calculatorForm.cacheWriteTokens" min="0" type="number" class="input" />
              </label>
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.cacheReadTokens') }}</span>
                <input v-model.number="calculatorForm.cacheReadTokens" min="0" type="number" class="input" />
              </label>
            </div>

            <div v-else class="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <label class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.contextTokens') }}</span>
                <input v-model.number="calculatorForm.inputTokens" min="0" type="number" class="input" />
              </label>
              <label v-if="calculatorForm.billingMode === BILLING_MODE_IMAGE" class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.imageCount') }}</span>
                <input v-model.number="calculatorForm.imageCount" min="0" type="number" class="input" />
              </label>
              <label v-else class="block">
                <span class="input-label">{{ t('modelMarketplace.calculator.requestCount') }}</span>
                <input v-model.number="calculatorForm.requestCount" min="0" type="number" class="input" />
              </label>
            </div>

            <div v-if="calculatorResult" class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.calculator.originalCost') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ formatCost(calculatorResult.selected?.baseCost ?? null) }}
                </div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
                <div class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('modelMarketplace.calculator.multipliedCost', { multiplier: calculatorResult.selected?.multiplier.toFixed(4) || '-' }) }}
                </div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ formatCost(calculatorResult.selected?.actualCost ?? null) }}
                </div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.calculator.lowestChannel') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ formatCost(calculatorResult.lowest?.actualCost ?? null) }}
                </div>
                <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ calculatorResult.lowest?.channelName || '-' }}
                </div>
              </div>
              <div class="rounded-lg bg-gray-50 p-3 dark:bg-dark-700/50">
                <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.calculator.highestChannel') }}</div>
                <div class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                  {{ formatCost(calculatorResult.highest?.actualCost ?? null) }}
                </div>
                <div class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ calculatorResult.highest?.channelName || '-' }}
                </div>
              </div>
            </div>

            <div v-else class="mt-4 rounded-lg bg-gray-50 p-3 text-sm text-gray-500 dark:bg-dark-700/50 dark:text-gray-400">
              {{ t('modelMarketplace.calculator.noResult') }}
            </div>
          </section>
        </div>
      </template>

      <template #table>
        <div class="space-y-3 md:hidden">
          <div
            v-if="loading"
            class="rounded-lg border border-gray-200 bg-white py-10 text-center dark:border-dark-700 dark:bg-dark-900"
          >
            <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
          </div>
          <div
            v-else-if="filteredRows.length === 0"
            class="rounded-lg border border-gray-200 bg-white px-4 py-12 text-center dark:border-dark-700 dark:bg-dark-900"
          >
            <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
            <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.empty') }}</p>
          </div>
          <template v-else>
            <article
              v-for="row in filteredRows"
              :key="row.key"
              class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
            >
              <button
                type="button"
                class="flex w-full items-start justify-between gap-3 text-left"
                @click="toggleExpanded(row.key)"
              >
                <div class="min-w-0">
                  <div class="mb-2 flex flex-wrap items-center gap-2">
                    <span
                      :class="[
                        'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                        platformBadgeClass(row.platform),
                      ]"
                    >
                      <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                      {{ platformLabel(row.platform) }}
                    </span>
                    <span class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('modelMarketplace.modelSubtitle', { count: row.entries.length }) }}
                    </span>
                  </div>
                  <h3 class="break-words text-sm font-semibold text-gray-900 dark:text-white">
                    {{ row.name }}
                  </h3>
                </div>
                <Icon
                  :name="expandedRows.has(row.key) ? 'chevronDown' : 'chevronRight'"
                  size="sm"
                  class="mt-1 flex-shrink-0 text-gray-400"
                />
              </button>

              <div class="mt-4 grid grid-cols-1 gap-2 text-xs sm:grid-cols-3">
                <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.columns.availability') }}</div>
                  <div class="mt-1 font-medium text-gray-900 dark:text-white">
                    {{ t('modelMarketplace.availability.channels', { count: row.channelNames.length }) }}
                  </div>
                  <div class="mt-0.5 text-gray-500 dark:text-gray-400">
                    {{ t('modelMarketplace.availability.groups', { count: row.groupCount }) }}
                  </div>
                </div>
                <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.columns.price') }}</div>
                  <div class="mt-1 font-medium text-gray-900 dark:text-white">
                    {{ priceSummary(row) }}
                  </div>
                  <div class="mt-0.5 text-gray-500 dark:text-gray-400">
                    {{ billingSummary(row) }}
                  </div>
                </div>
                <div class="rounded-md bg-gray-50 p-3 dark:bg-dark-800">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.columns.channels') }}</div>
                  <div class="mt-2 flex flex-wrap gap-1.5">
                    <span
                      v-for="channelName in row.channelNames.slice(0, 4)"
                      :key="channelName"
                      class="max-w-full truncate rounded-md bg-white px-2 py-1 text-gray-700 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:ring-dark-600"
                    >
                      {{ channelName }}
                    </span>
                    <span
                      v-if="row.channelNames.length > 4"
                      class="rounded-md bg-white px-2 py-1 text-gray-500 ring-1 ring-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:ring-dark-600"
                    >
                      +{{ row.channelNames.length - 4 }}
                    </span>
                  </div>
                </div>
              </div>

              <div v-if="expandedRows.has(row.key)" class="mt-4 space-y-3 border-t border-gray-100 pt-4 dark:border-dark-800">
                <div
                  v-for="entry in row.entries"
                  :key="`${entry.channelName}-${entry.platform}-${entry.model.name}`"
                  class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
                >
                  <div class="flex flex-col gap-3">
                    <div class="min-w-0">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="font-medium text-gray-900 dark:text-white">
                          {{ entry.channelName }}
                        </span>
                        <SupportedModelChip
                          :model="entry.model"
                          pricing-key-prefix="availableChannels.pricing"
                          :no-pricing-label="t('availableChannels.noPricing')"
                          :show-platform="false"
                          :platform-hint="entry.platform"
                        />
                      </div>
                      <p v-if="entry.channelDescription" class="mt-1 line-clamp-3 text-xs text-gray-500 dark:text-gray-400">
                        {{ entry.channelDescription }}
                      </p>
                    </div>

                    <div class="flex flex-wrap gap-1.5">
                      <GroupBadge
                        v-for="group in entry.groups"
                        :key="group.id"
                        :name="group.name"
                        :platform="group.platform as GroupPlatform"
                        :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                        :rate-multiplier="group.rate_multiplier"
                        :user-rate-multiplier="userGroupRates[group.id] ?? null"
                        always-show-rate
                      />
                      <span v-if="entry.groups.length === 0" class="text-xs text-gray-400">
                        {{ t('modelMarketplace.noGroups') }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </article>
          </template>
        </div>

        <div class="table-wrapper hidden md:block">
          <table class="w-full min-w-[960px] table-fixed border-collapse text-sm">
            <thead>
              <tr class="border-b border-gray-100 bg-gray-50/80 text-xs font-medium uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:bg-dark-800/80 dark:text-gray-400">
                <th class="w-[300px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.model') }}</th>
                <th class="w-[150px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.platform') }}</th>
                <th class="w-[140px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.availability') }}</th>
                <th class="w-[220px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.price') }}</th>
                <th class="px-4 py-3 text-left">{{ t('modelMarketplace.columns.channels') }}</th>
              </tr>
            </thead>

            <tbody v-if="loading">
              <tr>
                <td colspan="5" class="py-10 text-center">
                  <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
                </td>
              </tr>
            </tbody>

            <tbody v-else-if="filteredRows.length === 0">
              <tr>
                <td colspan="5" class="py-12 text-center">
                  <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.empty') }}</p>
                </td>
              </tr>
            </tbody>

            <tbody v-else>
              <template v-for="row in filteredRows" :key="row.key">
                <tr class="border-b border-gray-100 transition-colors hover:bg-gray-50/70 dark:border-dark-800 dark:hover:bg-dark-800/50">
                  <td class="px-4 py-3 align-top">
                    <button
                      type="button"
                      class="flex max-w-full items-start gap-2 text-left"
                      @click="toggleExpanded(row.key)"
                    >
                      <Icon
                        :name="expandedRows.has(row.key) ? 'chevronDown' : 'chevronRight'"
                        size="sm"
                        class="mt-0.5 flex-shrink-0 text-gray-400"
                      />
                      <span class="min-w-0">
                        <span class="block truncate font-medium text-gray-900 dark:text-white">
                          {{ row.name }}
                        </span>
                        <span class="mt-1 block truncate text-xs text-gray-500 dark:text-gray-400">
                          {{ t('modelMarketplace.modelSubtitle', { count: row.entries.length }) }}
                        </span>
                      </span>
                    </button>
                  </td>

                  <td class="px-4 py-3 align-top">
                    <span
                      :class="[
                        'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
                        platformBadgeClass(row.platform),
                      ]"
                    >
                      <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                      {{ platformLabel(row.platform) }}
                    </span>
                  </td>

                  <td class="px-4 py-3 align-top">
                    <div class="space-y-1 text-xs">
                      <div class="font-medium text-gray-900 dark:text-white">
                        {{ t('modelMarketplace.availability.channels', { count: row.channelNames.length }) }}
                      </div>
                      <div class="text-gray-500 dark:text-gray-400">
                        {{ t('modelMarketplace.availability.groups', { count: row.groupCount }) }}
                      </div>
                    </div>
                  </td>

                  <td class="px-4 py-3 align-top">
                    <div class="space-y-1 text-xs">
                      <div class="font-medium text-gray-900 dark:text-white">
                        {{ priceSummary(row) }}
                      </div>
                      <div class="text-gray-500 dark:text-gray-400">
                        {{ billingSummary(row) }}
                      </div>
                    </div>
                  </td>

                  <td class="px-4 py-3 align-top">
                    <div class="flex flex-wrap gap-1.5">
                      <span
                        v-for="channelName in row.channelNames"
                        :key="channelName"
                        class="rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-300"
                      >
                        {{ channelName }}
                      </span>
                    </div>
                  </td>
                </tr>

                <tr v-if="expandedRows.has(row.key)" class="border-b border-gray-100 bg-gray-50/50 dark:border-dark-800 dark:bg-dark-900/30">
                  <td colspan="5" class="px-4 py-4">
                    <div class="space-y-3">
                      <div
                        v-for="entry in row.entries"
                        :key="`${entry.channelName}-${entry.platform}-${entry.model.name}`"
                        class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800"
                      >
                        <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                          <div class="min-w-0">
                            <div class="flex flex-wrap items-center gap-2">
                              <span class="font-medium text-gray-900 dark:text-white">
                                {{ entry.channelName }}
                              </span>
                              <SupportedModelChip
                                :model="entry.model"
                                pricing-key-prefix="availableChannels.pricing"
                                :no-pricing-label="t('availableChannels.noPricing')"
                                :show-platform="false"
                                :platform-hint="entry.platform"
                              />
                            </div>
                            <p v-if="entry.channelDescription" class="mt-1 line-clamp-2 text-xs text-gray-500 dark:text-gray-400">
                              {{ entry.channelDescription }}
                            </p>
                          </div>

                          <div class="flex flex-wrap gap-1.5 lg:max-w-[55%] lg:justify-end">
                            <GroupBadge
                              v-for="group in entry.groups"
                              :key="group.id"
                              :name="group.name"
                              :platform="group.platform as GroupPlatform"
                              :subscription-type="(group.subscription_type || 'standard') as SubscriptionType"
                              :rate-multiplier="group.rate_multiplier"
                              :user-rate-multiplier="userGroupRates[group.id] ?? null"
                              always-show-rate
                            />
                            <span v-if="entry.groups.length === 0" class="text-xs text-gray-400">
                              {{ t('modelMarketplace.noGroups') }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from '@/components/channels/SupportedModelChip.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModel,
  type UserSupportedModelPricing,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST, BILLING_MODE_TOKEN, type BillingMode } from '@/constants/channel'
import { useAppStore } from '@/stores/app'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'
import { formatScaled } from '@/utils/pricing'

interface ModelMarketplaceEntry {
  channelName: string
  channelDescription: string
  platform: string
  groups: UserAvailableGroup[]
  model: UserSupportedModel
}

interface ModelMarketplaceRow {
  key: string
  name: string
  platform: string
  entries: ModelMarketplaceEntry[]
  channelNames: string[]
  groupCount: number
}

interface CalculatorCostRow {
  channelName: string
  baseCost: number
  actualCost: number
  multiplier: number
}

const { t } = useI18n()
const appStore = useAppStore()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const selectedPlatform = ref('')
const expandedRows = ref<Set<string>>(new Set())
const calculatorForm = reactive({
  modelKey: '',
  groupId: 0,
  channelName: '',
  billingMode: BILLING_MODE_TOKEN as BillingMode,
  inputTokens: 1000,
  outputTokens: 1000,
  cacheWriteTokens: 0,
  cacheReadTokens: 0,
  imageCount: 1,
  requestCount: 1,
})

const channelCount = computed(() => channels.value.length)

const platformOptions = computed(() => {
  const seen = new Set<string>()
  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      if (section.platform) seen.add(section.platform)
    }
  }
  return Array.from(seen).sort((a, b) => platformLabel(a).localeCompare(platformLabel(b)))
})

const modelRows = computed<ModelMarketplaceRow[]>(() => {
  const byModel = new Map<string, ModelMarketplaceEntry[]>()

  for (const channel of channels.value) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        const platform = model.platform || section.platform
        const key = `${platform.toLowerCase()}::${model.name.toLowerCase()}`
        const normalizedModel: UserSupportedModel = { ...model, platform }
        const entries = byModel.get(key) ?? []
        entries.push({
          channelName: channel.name,
          channelDescription: channel.description,
          platform,
          groups: section.groups,
          model: normalizedModel,
        })
        byModel.set(key, entries)
      }
    }
  }

  return Array.from(byModel.entries())
    .map(([key, entries]) => {
      const first = entries[0]
      const groupIds = new Set<number>()
      for (const entry of entries) {
        for (const group of entry.groups) {
          groupIds.add(group.id)
        }
      }
      return {
        key,
        name: first.model.name,
        platform: first.platform,
        entries: entries.sort((a, b) => a.channelName.localeCompare(b.channelName)),
        channelNames: Array.from(new Set(entries.map((entry) => entry.channelName))).sort(),
        groupCount: groupIds.size,
      }
    })
    .sort((a, b) => {
      const byName = a.name.localeCompare(b.name)
      return byName === 0 ? platformLabel(a.platform).localeCompare(platformLabel(b.platform)) : byName
    })
})

const filteredRows = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return modelRows.value.filter((row) => {
    if (selectedPlatform.value && row.platform !== selectedPlatform.value) return false
    if (!q) return true
    return (
      row.name.toLowerCase().includes(q) ||
      row.platform.toLowerCase().includes(q) ||
      platformLabel(row.platform).toLowerCase().includes(q) ||
      row.entries.some(
        (entry) =>
          entry.channelName.toLowerCase().includes(q) ||
          entry.channelDescription.toLowerCase().includes(q) ||
          entry.groups.some((group) => group.name.toLowerCase().includes(q)),
      )
    )
  })
})

const calculatorModelOptions = computed(() =>
  modelRows.value.map((row) => ({
    value: row.key,
    label: `${row.name} · ${platformLabel(row.platform)}`,
  })),
)

const calculatorRow = computed(() => modelRows.value.find((row) => row.key === calculatorForm.modelKey) || null)

const calculatorGroups = computed(() => {
  const row = calculatorRow.value
  if (!row) return []
  const groups = new Map<number, UserAvailableGroup>()
  for (const entry of row.entries) {
    for (const group of entry.groups) {
      groups.set(group.id, group)
    }
  }
  return Array.from(groups.values()).sort((a, b) => a.name.localeCompare(b.name))
})

const calculatorEntriesForGroup = computed(() => {
  const row = calculatorRow.value
  if (!row || !calculatorForm.groupId) return []
  return row.entries.filter((entry) => entry.groups.some((group) => group.id === calculatorForm.groupId))
})

const calculatorBillingModeOptions = computed(() => {
  const modes = new Set<BillingMode>()
  for (const entry of calculatorEntriesForGroup.value) {
    if (entry.model.pricing?.billing_mode) {
      modes.add(entry.model.pricing.billing_mode)
    }
  }
  const ordered: BillingMode[] = [BILLING_MODE_TOKEN, BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST]
  return ordered
    .filter((mode) => modes.has(mode))
    .map((mode) => ({ value: mode, label: billingModeLabel(mode) }))
})

const calculatorChannelOptions = computed(() => {
  const channelsForMode = calculatorEntriesForGroup.value
    .filter((entry) => entry.model.pricing?.billing_mode === calculatorForm.billingMode)
    .map((entry) => entry.channelName)
  return Array.from(new Set(channelsForMode)).sort()
})

const calculatorCostRows = computed<CalculatorCostRow[]>(() => {
  const selectedGroup = calculatorGroups.value.find((group) => group.id === calculatorForm.groupId)
  if (!selectedGroup) return []
  const multiplier = calculatorGroupMultiplier(selectedGroup)
  return calculatorEntriesForGroup.value
    .filter((entry) => entry.model.pricing?.billing_mode === calculatorForm.billingMode)
    .map((entry) => {
      const baseCost = calculatePricingCost(entry.model.pricing)
      if (baseCost === null) return null
      return {
        channelName: entry.channelName,
        baseCost,
        actualCost: baseCost * multiplier,
        multiplier,
      }
    })
    .filter((row): row is CalculatorCostRow => row !== null)
})

const calculatorResult = computed(() => {
  const rows = calculatorCostRows.value
  if (rows.length === 0) return null
  const selected = rows.find((row) => row.channelName === calculatorForm.channelName) || rows[0]
  const lowest = rows.reduce((best, row) => (row.actualCost < best.actualCost ? row : best), rows[0])
  const highest = rows.reduce((best, row) => (row.actualCost > best.actualCost ? row : best), rows[0])
  return { selected, lowest, highest }
})

function toggleExpanded(key: string) {
  const next = new Set(expandedRows.value)
  if (next.has(key)) {
    next.delete(key)
  } else {
    next.add(key)
  }
  expandedRows.value = next
}

function billingSummary(row: ModelMarketplaceRow): string {
  const modes = new Set(row.entries.map((entry) => entry.model.pricing?.billing_mode).filter(Boolean))
  if (modes.size === 0) return t('availableChannels.noPricing')
  const labels = Array.from(modes).map((mode) => billingModeLabel(mode as BillingMode))
  return labels.join(' / ')
}

function billingModeLabel(mode: BillingMode): string {
  if (mode === BILLING_MODE_TOKEN) return t('availableChannels.pricing.billingModeToken')
  if (mode === BILLING_MODE_PER_REQUEST) return t('availableChannels.pricing.billingModePerRequest')
  if (mode === BILLING_MODE_IMAGE) return t('availableChannels.pricing.billingModeImage')
  return String(mode)
}

function priceSummary(row: ModelMarketplaceRow): string {
  const prices = row.entries.map((entry) => entry.model.pricing).filter((pricing): pricing is UserSupportedModelPricing => pricing !== null)
  if (prices.length === 0) return t('availableChannels.noPricing')

  const tokenPrices = prices.filter((pricing) => pricing.billing_mode === BILLING_MODE_TOKEN)
  if (tokenPrices.length > 0) {
    const input = minNumber(tokenPrices.map((pricing) => pricing.input_price))
    const output = minNumber(tokenPrices.map((pricing) => pricing.output_price))
    return t('modelMarketplace.pricing.tokenSummary', {
      input: formatScaled(input, 1_000_000),
      output: formatScaled(output, 1_000_000),
    })
  }

  const requestPrices = prices.filter((pricing) => pricing.billing_mode === BILLING_MODE_PER_REQUEST)
  if (requestPrices.length > 0) {
    const price = minNumber(requestPrices.map((pricing) => pricing.per_request_price))
    return t('modelMarketplace.pricing.requestSummary', { price: formatScaled(price, 1) })
  }

  const imagePrices = prices.filter((pricing) => pricing.billing_mode === BILLING_MODE_IMAGE)
  if (imagePrices.length > 0) {
    const price = minNumber(imagePrices.map((pricing) => pricing.image_output_price ?? pricing.per_request_price))
    return t('modelMarketplace.pricing.imageSummary', { price: formatScaled(price, 1) })
  }

  return t('availableChannels.noPricing')
}

function minNumber(values: Array<number | null>): number | null {
  const numeric = values.filter((value): value is number => value !== null)
  if (numeric.length === 0) return null
  return Math.min(...numeric)
}

function calculatorGroupMultiplier(group: UserAvailableGroup): number {
  const multiplier = userGroupRates.value[group.id] ?? group.rate_multiplier
  return Number.isFinite(multiplier) && multiplier >= 0 ? multiplier : 1
}

function normalizeQuantity(value: number): number {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric < 0) return 0
  return numeric
}

function calculatorTotalContextTokens(): number {
  return Math.floor(normalizeQuantity(calculatorForm.inputTokens) + normalizeQuantity(calculatorForm.cacheReadTokens))
}

function matchingInterval(pricing: UserSupportedModelPricing) {
  const totalTokens = calculatorTotalContextTokens()
  return (pricing.intervals || []).find((interval) =>
    totalTokens > interval.min_tokens && (interval.max_tokens == null || totalTokens <= interval.max_tokens),
  ) || null
}

function calculatePricingCost(pricing: UserSupportedModelPricing | null): number | null {
  if (!pricing || pricing.billing_mode !== calculatorForm.billingMode) return null
  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    const interval = matchingInterval(pricing)
    const inputPrice = interval ? (interval.input_price ?? 0) : (pricing.input_price ?? 0)
    const outputPrice = interval ? (interval.output_price ?? 0) : (pricing.output_price ?? 0)
    const cacheWritePrice = interval ? (interval.cache_write_price ?? 0) : (pricing.cache_write_price ?? 0)
    const cacheReadPrice = interval ? (interval.cache_read_price ?? 0) : (pricing.cache_read_price ?? 0)
    return (
      normalizeQuantity(calculatorForm.inputTokens) * inputPrice +
      normalizeQuantity(calculatorForm.outputTokens) * outputPrice +
      normalizeQuantity(calculatorForm.cacheWriteTokens) * cacheWritePrice +
      normalizeQuantity(calculatorForm.cacheReadTokens) * cacheReadPrice
    )
  }

  const interval = matchingInterval(pricing)
  const intervalUnitPrice = interval?.per_request_price ?? null
  if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    const unitPrice = intervalUnitPrice ?? pricing.image_output_price ?? pricing.per_request_price
    if (unitPrice == null) return null
    return normalizeQuantity(calculatorForm.imageCount) * unitPrice
  }

  const unitPrice = intervalUnitPrice ?? pricing.per_request_price
  if (unitPrice == null) return null
  return normalizeQuantity(calculatorForm.requestCount) * unitPrice
}

function formatCost(value: number | null): string {
  if (value == null || !Number.isFinite(value)) return '-'
  if (value === 0) return '$0'
  const fixed = Math.abs(value) >= 1 ? value.toFixed(4) : value.toFixed(8)
  return `$${fixed.replace(/\.?0+$/, '')}`
}

function resetCalculatorInputs() {
  calculatorForm.inputTokens = 1000
  calculatorForm.outputTokens = 1000
  calculatorForm.cacheWriteTokens = 0
  calculatorForm.cacheReadTokens = 0
  calculatorForm.imageCount = 1
  calculatorForm.requestCount = 1
}

function syncCalculatorDefaults() {
  if (!calculatorForm.modelKey && modelRows.value.length > 0) {
    calculatorForm.modelKey = modelRows.value[0].key
    return
  }

  if (!calculatorGroups.value.some((group) => group.id === calculatorForm.groupId)) {
    calculatorForm.groupId = calculatorGroups.value[0]?.id ?? 0
    return
  }

  const modes = calculatorBillingModeOptions.value
  if (modes.length > 0 && !modes.some((option) => option.value === calculatorForm.billingMode)) {
    calculatorForm.billingMode = modes[0].value
    return
  }

  if (!calculatorChannelOptions.value.includes(calculatorForm.channelName)) {
    calculatorForm.channelName = calculatorChannelOptions.value[0] || ''
  }
}

async function loadChannels() {
  loading.value = true
  try {
    const [list, rates] = await Promise.all([
      userChannelsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
        console.error('Failed to load user group rates:', err)
        return {} as Record<number, number>
      }),
    ])
    channels.value = list
    userGroupRates.value = rates
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

watch([modelRows, calculatorGroups, calculatorBillingModeOptions, calculatorChannelOptions], syncCalculatorDefaults)

onMounted(loadChannels)
</script>
