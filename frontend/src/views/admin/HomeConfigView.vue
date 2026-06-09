<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-6">
      <section class="card overflow-hidden">
        <div class="border-b border-gray-100 p-5 dark:border-dark-700 sm:p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <div class="mb-3 inline-flex h-11 w-11 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
                <Icon name="home" size="lg" />
              </div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
                {{ t('admin.homeConfig.title') }}
              </h1>
              <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-gray-400">
                {{ t('admin.homeConfig.description') }}
              </p>
            </div>
            <div class="flex flex-col gap-2 sm:flex-row">
              <a href="/home" target="_blank" rel="noopener noreferrer" class="btn btn-secondary">
                <Icon name="externalLink" size="sm" />
                {{ t('admin.homeConfig.preview') }}
              </a>
              <button class="btn btn-primary" :disabled="saving || loading" @click="save">
                <Icon v-if="!saving" name="check" size="sm" />
                {{ saving ? t('common.saving') : t('common.save') }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <div v-if="loading" class="card flex justify-center p-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <section class="card space-y-5 p-5">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.homeConfig.basic.title') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.homeConfig.basic.description') }}</p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.homeConfig.basic.siteName') }}</label>
              <input v-model.trim="siteForm.site_name" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.basic.subtitle') }}</label>
              <input v-model.trim="siteForm.site_subtitle" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.basic.docUrl') }}</label>
              <input v-model.trim="siteForm.doc_url" class="input" placeholder="https://docs.example.com" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.basic.apiBaseUrl') }}</label>
              <input v-model.trim="siteForm.api_base_url" class="input" placeholder="https://api.example.com" />
            </div>
            <div class="md:col-span-2">
              <label class="input-label">{{ t('admin.homeConfig.basic.contactInfo') }}</label>
              <input v-model.trim="siteForm.contact_info" class="input" />
            </div>
            <div class="md:col-span-2">
              <label class="input-label">{{ t('admin.homeConfig.basic.homeContent') }}</label>
              <textarea v-model="siteForm.home_content" class="input min-h-28 font-mono text-sm" :placeholder="t('admin.homeConfig.basic.homeContentPlaceholder')"></textarea>
              <p class="mt-1 text-xs text-amber-600 dark:text-amber-400">{{ t('admin.homeConfig.basic.homeContentHint') }}</p>
            </div>
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <SectionHeader :title="t('admin.homeConfig.nav.title')" :description="t('admin.homeConfig.nav.description')" @add="addNavItem" />
          <div class="space-y-3">
            <div v-for="(item, index) in config.nav_items" :key="index" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="grid gap-3 md:grid-cols-[1fr_1.4fr_auto_auto] md:items-end">
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.label') }}</label>
                  <input v-model.trim="item.label" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.url') }}</label>
                  <input v-model.trim="item.url" class="input" placeholder="#features 或 https://..." />
                </div>
                <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="item.visible" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                  {{ t('admin.homeConfig.fields.visible') }}
                </label>
                <button class="btn btn-danger btn-sm" @click="config.nav_items.splice(index, 1)">{{ t('common.delete') }}</button>
              </div>
            </div>
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.homeConfig.hero.title') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.homeConfig.hero.description') }}</p>
          </div>
          <div class="grid gap-4 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.badge') }}</label>
              <input v-model.trim="config.hero_badge" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.terminalTitle') }}</label>
              <input v-model.trim="config.terminal_title" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.mainTitle') }}</label>
              <input v-model.trim="config.hero_title" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.highlight') }}</label>
              <input v-model.trim="config.hero_highlight" class="input" />
            </div>
            <div class="md:col-span-2">
              <label class="input-label">{{ t('admin.homeConfig.hero.copy') }}</label>
              <textarea v-model="config.hero_description" class="input min-h-24"></textarea>
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.primaryCta') }}</label>
              <input v-model.trim="config.primary_cta_label" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.primaryUrl') }}</label>
              <input v-model.trim="config.primary_cta_url" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.secondaryCta') }}</label>
              <input v-model.trim="config.secondary_cta_label" class="input" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.homeConfig.hero.secondaryUrl') }}</label>
              <input v-model.trim="config.secondary_cta_url" class="input" />
            </div>
            <div class="md:col-span-2">
              <label class="input-label">{{ t('admin.homeConfig.hero.terminalLines') }}</label>
              <textarea v-model="terminalLinesText" class="input min-h-28 font-mono text-sm"></textarea>
            </div>
          </div>
        </section>

        <section class="grid gap-6 lg:grid-cols-2">
          <div class="card space-y-5 p-5">
            <SectionHeader :title="t('admin.homeConfig.stats.title')" :description="t('admin.homeConfig.stats.description')" @add="addStat" />
            <SimpleListEditor
              v-model:items="config.stats"
              value-key="value"
              :value-label="t('admin.homeConfig.fields.value')"
              :label-label="t('admin.homeConfig.fields.label')"
            />
          </div>
          <div class="card space-y-5 p-5">
            <SectionHeader :title="t('admin.homeConfig.info.title')" :description="t('admin.homeConfig.info.description')" @add="addInfo" />
            <InfoListEditor v-model:items="config.info_items" />
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <SectionHeader :title="t('admin.homeConfig.features.title')" :description="t('admin.homeConfig.features.description')" @add="addFeature" />
          <div class="grid gap-4 lg:grid-cols-2">
            <div v-for="(item, index) in config.features" :key="index" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.title') }}</label>
                  <input v-model.trim="item.title" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.icon') }}</label>
                  <select v-model="item.icon" class="input">
                    <option v-for="icon in iconOptions" :key="icon" :value="icon">{{ icon }}</option>
                  </select>
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.tag') }}</label>
                  <input v-model.trim="item.tag" class="input" />
                </div>
                <label class="flex items-center gap-2 pt-7 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="item.visible" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                  {{ t('admin.homeConfig.fields.visible') }}
                </label>
                <div class="sm:col-span-2">
                  <label class="input-label">{{ t('admin.homeConfig.fields.description') }}</label>
                  <textarea v-model="item.description" class="input min-h-20"></textarea>
                </div>
              </div>
              <div class="mt-3 flex justify-end">
                <button class="btn btn-danger btn-sm" @click="config.features.splice(index, 1)">{{ t('common.delete') }}</button>
              </div>
            </div>
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <SectionHeader :title="t('admin.homeConfig.models.title')" :description="t('admin.homeConfig.models.description')" @add="addModel" />
          <div class="grid gap-4 lg:grid-cols-2">
            <div v-for="(item, index) in config.models" :key="index" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.name') }}</label>
                  <input v-model.trim="item.name" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.models.provider') }}</label>
                  <input v-model.trim="item.provider" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.models.price') }}</label>
                  <input v-model.trim="item.price" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.models.status') }}</label>
                  <input v-model.trim="item.status" class="input" />
                </div>
                <div class="sm:col-span-2">
                  <label class="input-label">{{ t('admin.homeConfig.fields.description') }}</label>
                  <textarea v-model="item.description" class="input min-h-20"></textarea>
                </div>
              </div>
              <div class="mt-3 flex items-center justify-between">
                <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="item.visible" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                  {{ t('admin.homeConfig.fields.visible') }}
                </label>
                <button class="btn btn-danger btn-sm" @click="config.models.splice(index, 1)">{{ t('common.delete') }}</button>
              </div>
            </div>
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <SectionHeader :title="t('admin.homeConfig.pricing.title')" :description="t('admin.homeConfig.pricing.description')" @add="addPricing" />
          <div class="grid gap-4 lg:grid-cols-3">
            <div v-for="(item, index) in config.pricing_items" :key="index" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="space-y-3">
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.name') }}</label>
                  <input v-model.trim="item.name" class="input" />
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="input-label">{{ t('admin.homeConfig.pricing.price') }}</label>
                    <input v-model.trim="item.price" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ t('admin.homeConfig.pricing.unit') }}</label>
                    <input v-model.trim="item.unit" class="input" />
                  </div>
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.description') }}</label>
                  <textarea v-model="item.description" class="input min-h-20"></textarea>
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.pricing.features') }}</label>
                  <textarea :value="(item.features || []).join('\n')" class="input min-h-24" @input="item.features = textareaLines($event)"></textarea>
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label class="input-label">{{ t('admin.homeConfig.pricing.ctaLabel') }}</label>
                    <input v-model.trim="item.cta_label" class="input" />
                  </div>
                  <div>
                    <label class="input-label">{{ t('admin.homeConfig.pricing.ctaUrl') }}</label>
                    <input v-model.trim="item.cta_url" class="input" />
                  </div>
                </div>
                <div class="flex items-center justify-between">
                  <div class="flex flex-wrap gap-4">
                    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                      <input v-model="item.highlighted" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                      {{ t('admin.homeConfig.pricing.highlighted') }}
                    </label>
                    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                      <input v-model="item.visible" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                      {{ t('admin.homeConfig.fields.visible') }}
                    </label>
                  </div>
                  <button class="btn btn-danger btn-sm" @click="config.pricing_items.splice(index, 1)">{{ t('common.delete') }}</button>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="card space-y-5 p-5">
          <SectionHeader :title="t('admin.homeConfig.customSections.title')" :description="t('admin.homeConfig.customSections.description')" @add="addCustomSection" />
          <div class="space-y-4">
            <div v-for="(section, sectionIndex) in config.custom_sections" :key="sectionIndex" class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <div class="grid gap-3 md:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.customSections.eyebrow') }}</label>
                  <input v-model.trim="section.eyebrow" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.fields.title') }}</label>
                  <input v-model.trim="section.title" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.customSections.layout') }}</label>
                  <select v-model="section.layout" class="input">
                    <option value="cards">{{ t('admin.homeConfig.customSections.layouts.cards') }}</option>
                    <option value="metrics">{{ t('admin.homeConfig.customSections.layouts.metrics') }}</option>
                    <option value="text">{{ t('admin.homeConfig.customSections.layouts.text') }}</option>
                  </select>
                </div>
                <label class="flex items-center gap-2 pt-7 text-sm text-gray-700 dark:text-gray-300">
                  <input v-model="section.visible" type="checkbox" class="rounded border-gray-300 text-primary-600" />
                  {{ t('admin.homeConfig.fields.visible') }}
                </label>
                <div class="md:col-span-2">
                  <label class="input-label">{{ t('admin.homeConfig.fields.description') }}</label>
                  <textarea v-model="section.description" class="input min-h-20"></textarea>
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.pricing.ctaLabel') }}</label>
                  <input v-model.trim="section.cta_label" class="input" />
                </div>
                <div>
                  <label class="input-label">{{ t('admin.homeConfig.pricing.ctaUrl') }}</label>
                  <input v-model.trim="section.cta_url" class="input" />
                </div>
              </div>

              <div class="mt-5 rounded-lg bg-gray-50 p-4 dark:bg-dark-800/60">
                <div class="mb-3 flex items-center justify-between gap-3">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.homeConfig.customSections.items') }}</h3>
                  <button class="btn btn-secondary btn-sm" @click="addCustomSectionItem(sectionIndex)">{{ t('common.add') }}</button>
                </div>
                <div class="space-y-3">
                  <div v-for="(item, itemIndex) in section.items || []" :key="itemIndex" class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-900">
                    <div class="grid gap-3 md:grid-cols-[1fr_1fr_auto] md:items-end">
                      <div>
                        <label class="input-label">{{ t('admin.homeConfig.fields.label') }}</label>
                        <input v-model.trim="item.label" class="input" />
                      </div>
                      <div>
                        <label class="input-label">{{ t('admin.homeConfig.fields.value') }}</label>
                        <input v-model.trim="item.value" class="input" />
                      </div>
                      <button class="btn btn-danger btn-sm" @click="section.items?.splice(itemIndex, 1)">{{ t('common.delete') }}</button>
                    </div>
                    <div class="mt-3">
                      <label class="input-label">{{ t('admin.homeConfig.fields.description') }}</label>
                      <textarea v-model="item.description" class="input min-h-16"></textarea>
                    </div>
                  </div>
                </div>
              </div>

              <div class="mt-3 flex justify-end">
                <button class="btn btn-danger btn-sm" @click="config.custom_sections.splice(sectionIndex, 1)">{{ t('common.delete') }}</button>
              </div>
            </div>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { SystemSettings } from '@/api/admin/settings'
import type { HomeConfig, HomeCustomSectionItem, HomeFeatureItem, HomeInfoItem, HomeModelItem, HomeNavItem, HomePricingItem, HomeStatItem } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { stripDefaultHomeTextStrings } from '@/utils/homeConfigDefaults'

type EditableHomeConfig = Required<Pick<HomeConfig,
  'nav_items' | 'stats' | 'terminal_lines' | 'features' | 'models' | 'pricing_items' | 'info_items' | 'custom_sections'
>> & Omit<HomeConfig, 'nav_items' | 'stats' | 'terminal_lines' | 'features' | 'models' | 'pricing_items' | 'info_items' | 'custom_sections'>

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const loadedSettings = ref<SystemSettings | null>(null)

const siteForm = reactive({
  site_name: '',
  site_subtitle: '',
  doc_url: '',
  api_base_url: '',
  contact_info: '',
  home_content: ''
})

const config = reactive<EditableHomeConfig>(emptyHomeConfig())
const iconOptions = ['server', 'shield', 'chart', 'database', 'bolt', 'key', 'globe', 'terminal', 'cloud', 'cpu', 'calculator', 'brain']

const terminalLinesText = computed({
  get: () => config.terminal_lines.join('\n'),
  set: (value: string) => {
    config.terminal_lines = value.split('\n').map((line) => line.trim()).filter(Boolean)
  }
})

const SectionHeader = defineComponent({
  props: {
    title: { type: String, required: true },
    description: { type: String, default: '' }
  },
  emits: ['add'],
  setup(props, { emit }) {
    return () => h('div', { class: 'flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between' }, [
      h('div', [
        h('h2', { class: 'text-lg font-semibold text-gray-900 dark:text-white' }, props.title),
        props.description ? h('p', { class: 'mt-1 text-sm text-gray-500 dark:text-gray-400' }, props.description) : null
      ]),
      h('button', { class: 'btn btn-secondary btn-sm', onClick: () => emit('add') }, t('common.add'))
    ])
  }
})

const SimpleListEditor = defineComponent({
  props: {
    items: { type: Array as PropType<HomeStatItem[]>, required: true },
    valueKey: { type: String, required: true },
    valueLabel: { type: String, required: true },
    labelLabel: { type: String, required: true }
  },
  emits: ['update:items'],
  setup(props, { emit }) {
    const remove = (index: number) => {
      const next = [...props.items]
      next.splice(index, 1)
      emit('update:items', next)
    }
    return () => h('div', { class: 'space-y-3' }, props.items.map((item, index) =>
      h('div', { class: 'rounded-lg border border-gray-200 p-4 dark:border-dark-700' }, [
        h('div', { class: 'grid gap-3 sm:grid-cols-[1fr_1fr_auto_auto] sm:items-end' }, [
          h('label', { class: 'block' }, [
            h('span', { class: 'input-label' }, props.valueLabel),
            h('input', {
              class: 'input',
              value: item.value,
              onInput: (event: Event) => item.value = (event.target as HTMLInputElement).value
            })
          ]),
          h('label', { class: 'block' }, [
            h('span', { class: 'input-label' }, props.labelLabel),
            h('input', {
              class: 'input',
              value: item.label,
              onInput: (event: Event) => item.label = (event.target as HTMLInputElement).value
            })
          ]),
          h('label', { class: 'flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300' }, [
            h('input', {
              type: 'checkbox',
              class: 'rounded border-gray-300 text-primary-600',
              checked: item.visible !== false,
              onChange: (event: Event) => item.visible = (event.target as HTMLInputElement).checked
            }),
            t('admin.homeConfig.fields.visible')
          ]),
          h('button', { class: 'btn btn-danger btn-sm', onClick: () => remove(index) }, t('common.delete'))
        ])
      ])
    ))
  }
})

const InfoListEditor = defineComponent({
  props: {
    items: { type: Array as PropType<HomeInfoItem[]>, required: true }
  },
  emits: ['update:items'],
  setup(props, { emit }) {
    const remove = (index: number) => {
      const next = [...props.items]
      next.splice(index, 1)
      emit('update:items', next)
    }
    return () => h('div', { class: 'space-y-3' }, props.items.map((item, index) =>
      h('div', { class: 'rounded-lg border border-gray-200 p-4 dark:border-dark-700' }, [
        h('div', { class: 'grid gap-3 sm:grid-cols-2' }, [
          inputNode(t('admin.homeConfig.fields.label'), item.label, (value) => item.label = value),
          inputNode(t('admin.homeConfig.fields.value'), item.value, (value) => item.value = value),
          h('label', { class: 'sm:col-span-2 block' }, [
            h('span', { class: 'input-label' }, t('admin.homeConfig.fields.description')),
            h('textarea', {
              class: 'input min-h-20',
              value: item.description || '',
              onInput: (event: Event) => item.description = (event.target as HTMLTextAreaElement).value
            })
          ])
        ]),
        h('div', { class: 'mt-3 flex items-center justify-between' }, [
          h('label', { class: 'flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300' }, [
            h('input', {
              type: 'checkbox',
              class: 'rounded border-gray-300 text-primary-600',
              checked: item.visible !== false,
              onChange: (event: Event) => item.visible = (event.target as HTMLInputElement).checked
            }),
            t('admin.homeConfig.fields.visible')
          ]),
          h('button', { class: 'btn btn-danger btn-sm', onClick: () => remove(index) }, t('common.delete'))
        ])
      ])
    ))
  }
})

function inputNode(label: string, value: string, onInput: (value: string) => void) {
  return h('label', { class: 'block' }, [
    h('span', { class: 'input-label' }, label),
    h('input', {
      class: 'input',
      value,
      onInput: (event: Event) => onInput((event.target as HTMLInputElement).value)
    })
  ])
}

function emptyHomeConfig(): EditableHomeConfig {
  return {
    nav_items: [],
    hero_badge: '',
    hero_title: '',
    hero_highlight: '',
    hero_description: '',
    primary_cta_label: '',
    primary_cta_url: '',
    secondary_cta_label: '',
    secondary_cta_url: '',
    stats: [],
    terminal_title: '',
    terminal_lines: [],
    features_title: '',
    features_description: '',
    features: [],
    models_title: '',
    models_description: '',
    models: [],
    pricing_title: '',
    pricing_description: '',
    pricing_items: [],
    info_title: '',
    info_description: '',
    info_items: [],
    custom_sections: []
  }
}

function applyHomeConfig(raw: HomeConfig | null | undefined) {
  const next = normalizeHomeConfig(raw)
  Object.assign(config, next)
}

function normalizeHomeConfig(raw: HomeConfig | null | undefined): EditableHomeConfig {
  const base = defaultAdminHomeConfig()
  const source = raw && Object.keys(raw).length > 0 ? raw : base
  return {
    ...base,
    ...source,
    hero_badge: resolveAdminText(base.hero_badge, source.hero_badge),
    hero_title: resolveAdminText(base.hero_title, source.hero_title),
    hero_highlight: resolveAdminText(base.hero_highlight, source.hero_highlight),
    hero_description: resolveAdminText(base.hero_description, source.hero_description),
    primary_cta_label: resolveAdminText(base.primary_cta_label, source.primary_cta_label),
    primary_cta_url: resolveAdminText(base.primary_cta_url, source.primary_cta_url),
    secondary_cta_label: resolveAdminText(base.secondary_cta_label, source.secondary_cta_label),
    secondary_cta_url: resolveAdminText(base.secondary_cta_url, source.secondary_cta_url),
    terminal_title: resolveAdminText(base.terminal_title, source.terminal_title),
    features_title: resolveAdminText(base.features_title, source.features_title),
    features_description: resolveAdminText(base.features_description, source.features_description),
    models_title: resolveAdminText(base.models_title, source.models_title),
    models_description: resolveAdminText(base.models_description, source.models_description),
    pricing_title: resolveAdminText(base.pricing_title, source.pricing_title),
    pricing_description: resolveAdminText(base.pricing_description, source.pricing_description),
    info_title: resolveAdminText(base.info_title, source.info_title),
    info_description: resolveAdminText(base.info_description, source.info_description),
    nav_items: Array.isArray(source.nav_items) ? normalizeNavItems(source.nav_items, base.nav_items) : base.nav_items,
    stats: Array.isArray(source.stats) ? normalizeStats(source.stats, base.stats) : base.stats,
    terminal_lines: Array.isArray(source.terminal_lines) ? normalizeStringList(source.terminal_lines, base.terminal_lines) : base.terminal_lines,
    features: Array.isArray(source.features) ? normalizeFeatures(source.features, base.features) : base.features,
    models: Array.isArray(source.models) ? normalizeModels(source.models, base.models) : base.models,
    pricing_items: Array.isArray(source.pricing_items) ? normalizePricingItems(source.pricing_items, base.pricing_items) : base.pricing_items,
    info_items: Array.isArray(source.info_items) ? normalizeInfoItems(source.info_items, base.info_items) : base.info_items,
    custom_sections: Array.isArray(source.custom_sections) ? normalizeCustomSections(source.custom_sections, base.custom_sections) : base.custom_sections
  }
}

function defaultAdminHomeConfig(): EditableHomeConfig {
  return {
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
    primary_cta_label: t('home.getStarted'),
    primary_cta_url: '/login',
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
        cta_url: siteForm.doc_url || '#info',
        visible: true
      }
    ],
    info_title: t('home.infoSection.title'),
    info_description: t('home.infoSection.description'),
    info_items: [
      { label: t('home.infoSection.apiEndpoint'), value: siteForm.api_base_url || window.location.origin, description: t('home.infoSection.apiEndpointDesc'), visible: true },
      { label: t('home.infoSection.billing'), value: t('home.infoSection.billingValue'), description: t('home.infoSection.billingDesc'), visible: true },
      { label: t('home.infoSection.security'), value: t('home.infoSection.securityValue'), description: t('home.infoSection.securityDesc'), visible: true },
      { label: t('home.infoSection.contact'), value: siteForm.contact_info || t('home.infoSection.contactValue'), description: t('home.infoSection.contactDesc'), visible: true }
    ],
    custom_sections: [
      {
        eyebrow: t('admin.homeConfig.customSections.defaultEyebrow'),
        title: t('admin.homeConfig.customSections.defaultTitle'),
        description: t('admin.homeConfig.customSections.defaultDescription'),
        layout: 'cards',
        items: [
          { label: 'SDK', value: 'OpenAI Compatible', description: t('home.features.compatibleDesc'), visible: true },
          { label: 'Routing', value: 'Smart Failover', description: t('home.features.multiAccountDesc'), visible: true },
          { label: 'Ops', value: 'Usage Insights', description: t('home.features.logsDesc'), visible: true }
        ],
        visible: false
      }
    ]
  }
}

function normalizeList<T extends { visible?: boolean }>(items: T[] | undefined, fallback: () => T): T[] {
  if (!Array.isArray(items)) return []
  return items.map((item) => ({ ...fallback(), ...item, visible: item.visible !== false }))
}

function resolveAdminText(defaultValue: string | undefined, value: string | undefined): string {
  return value && value.trim().length > 0 ? value : defaultValue || ''
}

function normalizeStringList(items: string[] | undefined, defaults: string[]): string[] {
  if (!Array.isArray(items)) return defaults
  return items.map((item, index) => resolveAdminText(defaults[index], item))
}

function normalizeNavItems(items: HomeNavItem[] | undefined, defaults: HomeNavItem[]): HomeNavItem[] {
  return normalizeList<HomeNavItem>(items, () => ({ label: '', url: '#top', visible: true }))
    .map((item, index) => ({
      ...item,
      label: resolveAdminText(defaults[index]?.label, item.label),
      url: resolveAdminText(defaults[index]?.url, item.url)
    }))
}

function normalizeStats(items: HomeStatItem[] | undefined, defaults: HomeStatItem[]): HomeStatItem[] {
  return normalizeList<HomeStatItem>(items, () => ({ value: '', label: '', visible: true }))
    .map((item, index) => ({
      ...item,
      value: resolveAdminText(defaults[index]?.value, item.value),
      label: resolveAdminText(defaults[index]?.label, item.label)
    }))
}

function normalizeFeatures(items: HomeFeatureItem[] | undefined, defaults: HomeFeatureItem[]): HomeFeatureItem[] {
  return normalizeList<HomeFeatureItem>(items, () => ({ title: '', description: '', icon: 'server', tag: '', visible: true }))
    .map((item, index) => ({
      ...item,
      title: resolveAdminText(defaults[index]?.title, item.title),
      description: resolveAdminText(defaults[index]?.description, item.description),
      tag: resolveAdminText(defaults[index]?.tag, item.tag)
    }))
}

function normalizeModels(items: HomeModelItem[] | undefined, defaults: HomeModelItem[]): HomeModelItem[] {
  return normalizeList<HomeModelItem>(items, () => ({ name: '', provider: '', description: '', price: '', status: '', visible: true }))
    .map((item, index) => ({
      ...item,
      name: resolveAdminText(defaults[index]?.name, item.name),
      provider: resolveAdminText(defaults[index]?.provider, item.provider),
      description: resolveAdminText(defaults[index]?.description, item.description),
      price: resolveAdminText(defaults[index]?.price, item.price),
      status: resolveAdminText(defaults[index]?.status, item.status)
    }))
}

function normalizePricingItems(items: HomePricingItem[] | undefined, defaults: HomePricingItem[]): HomePricingItem[] {
  return normalizeList<HomePricingItem>(items, () => ({ name: '', price: '', unit: '', description: '', features: [], cta_label: '', cta_url: '', highlighted: false, visible: true }))
    .map((item, index) => ({
      ...item,
      name: resolveAdminText(defaults[index]?.name, item.name),
      price: resolveAdminText(defaults[index]?.price, item.price),
      unit: resolveAdminText(defaults[index]?.unit, item.unit),
      description: resolveAdminText(defaults[index]?.description, item.description),
      features: normalizeStringList(item.features, defaults[index]?.features || []),
      cta_label: resolveAdminText(defaults[index]?.cta_label, item.cta_label),
      cta_url: resolveAdminText(defaults[index]?.cta_url, item.cta_url)
    }))
}

function normalizeInfoItems(items: HomeInfoItem[] | undefined, defaults: HomeInfoItem[]): HomeInfoItem[] {
  return normalizeList<HomeInfoItem>(items, () => ({ label: '', value: '', description: '', visible: true }))
    .map((item, index) => ({
      ...item,
      label: resolveAdminText(defaults[index]?.label, item.label),
      value: resolveAdminText(defaults[index]?.value, item.value),
      description: resolveAdminText(defaults[index]?.description, item.description)
    }))
}

function normalizeCustomSections(items: HomeCustomSectionItem[] | undefined, defaults: HomeCustomSectionItem[] = []): HomeCustomSectionItem[] {
  if (!Array.isArray(items)) return []
  return items.map((section, index) => {
    const fallback = defaults[index]
    return {
      ...section,
      eyebrow: resolveAdminText(fallback?.eyebrow, section.eyebrow),
      title: resolveAdminText(fallback?.title, section.title),
      description: resolveAdminText(fallback?.description, section.description),
      layout: section.layout || 'cards',
      cta_label: section.cta_label || '',
      cta_url: section.cta_url || '',
      visible: section.visible !== false,
      items: normalizeInfoItems(section.items, fallback?.items || [])
    }
  })
}

function toHomeConfigPayload(): HomeConfig {
  return stripDefaultHomeTextStrings(JSON.parse(JSON.stringify(config)) as HomeConfig)
}

function addNavItem() {
  config.nav_items.push({ label: t('admin.homeConfig.nav.newItem'), url: '#features', visible: true })
}

function addStat() {
  config.stats.push({ value: '99.9%', label: t('admin.homeConfig.stats.newItem'), visible: true })
}

function addInfo() {
  config.info_items.push({ label: t('admin.homeConfig.info.newItem'), value: '', description: '', visible: true })
}

function addFeature() {
  config.features.push({ title: t('admin.homeConfig.features.newItem'), description: '', icon: 'server', tag: '', visible: true })
}

function addModel() {
  config.models.push({ name: t('admin.homeConfig.models.newItem'), provider: '', description: '', price: '', status: t('home.providers.supported'), visible: true })
}

function addPricing() {
  config.pricing_items.push({ name: t('admin.homeConfig.pricing.newItem'), price: '', unit: '', description: '', features: [], cta_label: '', cta_url: '', highlighted: false, visible: true })
}

function addCustomSection() {
  config.custom_sections.push({
    eyebrow: '',
    title: t('admin.homeConfig.customSections.newItem'),
    description: '',
    layout: 'cards',
    items: [],
    cta_label: '',
    cta_url: '',
    visible: true
  })
}

function addCustomSectionItem(sectionIndex: number) {
  const section = config.custom_sections[sectionIndex]
  if (!section) return
  if (!Array.isArray(section.items)) section.items = []
  section.items.push({ label: t('admin.homeConfig.info.newItem'), value: '', description: '', visible: true })
}

function textareaLines(event: Event): string[] {
  return (event.target as HTMLTextAreaElement).value.split('\n').map((line) => line.trim()).filter(Boolean)
}

async function load() {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getSettings()
    loadedSettings.value = settings
    siteForm.site_name = settings.site_name || 'AySub'
    siteForm.site_subtitle = settings.site_subtitle || ''
    siteForm.doc_url = settings.doc_url || ''
    siteForm.api_base_url = settings.api_base_url || ''
    siteForm.contact_info = settings.contact_info || ''
    siteForm.home_content = settings.home_content || ''
    applyHomeConfig(settings.home_config)
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.homeConfig.loadFailed')))
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!loadedSettings.value) return
  saving.value = true
  try {
    const payload = {
      ...loadedSettings.value,
      site_name: siteForm.site_name,
      site_subtitle: siteForm.site_subtitle,
      doc_url: siteForm.doc_url,
      api_base_url: siteForm.api_base_url,
      contact_info: siteForm.contact_info,
      home_content: siteForm.home_content,
      home_config: toHomeConfigPayload()
    }
    const updated = await adminAPI.settings.updateSettings(payload)
    loadedSettings.value = updated
    appStore.clearPublicSettingsCache()
    await appStore.fetchPublicSettings(true)
    appStore.showSuccess(t('admin.homeConfig.saveSuccess'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.homeConfig.saveFailed')))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
