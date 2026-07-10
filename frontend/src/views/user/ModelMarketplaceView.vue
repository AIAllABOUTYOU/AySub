<template>
  <component :is="pageShell" :class="publicShellClass">
    <template v-if="!isAuthenticated">
      <div class="home-grid" aria-hidden="true"></div>

      <!-- Public Header -->
      <PublicHeader :current-path="currentPath" home-url="/home" dashboard-url="/console" />
    </template>

    <main :id="!isAuthenticated ? 'top' : undefined" :class="mainShellClass">
      <div :class="marketplaceContainerClass">
        <header class="marketplace-hero marketplace-panel flex flex-col gap-5 rounded-lg border border-gray-200 bg-white px-4 py-5 shadow-sm dark:border-dark-700 dark:bg-dark-800 sm:px-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <div class="mb-3 inline-flex items-center gap-2 rounded-full border border-gray-200 bg-white px-3 py-1 text-xs font-medium text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
              <Icon name="sparkles" size="xs" class="text-cyan-500" />
              {{ t('modelMarketplace.publicEyebrow') }}
            </div>
            <h1 class="text-2xl font-bold tracking-normal text-gray-950 dark:text-white sm:text-3xl">
              {{ t('modelMarketplace.title') }}
            </h1>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ isAuthenticated ? t('modelMarketplace.description') : t('modelMarketplace.publicDescription') }}
            </p>
          </div>

          <div class="marketplace-stats-grid grid grid-cols-3 gap-2 text-xs text-gray-500 dark:text-gray-400 sm:min-w-[390px]">
            <div class="marketplace-stat">
              <span>{{ t('modelMarketplace.stats.models', { count: modelRows.length }) }}</span>
            </div>
            <div class="marketplace-stat">
              <span>{{ t('modelMarketplace.stats.channels', { count: channelCount }) }}</span>
            </div>
            <div class="marketplace-stat">
              <span>{{ t('modelMarketplace.stats.platforms', { count: platformOptions.length }) }}</span>
            </div>
          </div>
        </header>

        <section class="marketplace-panel marketplace-toolbar rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-center">
            <div class="relative min-w-0 flex-1">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('modelMarketplace.searchPlaceholder')"
                class="h-10 w-full rounded-md border border-gray-200 bg-gray-50 pl-10 pr-4 text-sm text-gray-900 outline-none transition focus:border-cyan-400 focus:bg-white focus:ring-4 focus:ring-cyan-100 dark:border-dark-700 dark:bg-dark-900/70 dark:text-white dark:focus:border-cyan-500 dark:focus:ring-cyan-950"
              />
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <div class="inline-flex h-9 items-center gap-2 rounded-md border border-gray-200 px-3 text-sm text-gray-600 dark:border-dark-700 dark:text-gray-300">
                <span>{{ t('modelMarketplace.toolbar.multiplier', '倍率') }}</span>
                <button
                  type="button"
                  data-testid="marketplace-multiplier-toggle"
                  class="relative h-5 w-9 rounded-full transition"
                  :class="showMultiplier ? 'bg-cyan-500' : 'bg-gray-200 dark:bg-dark-600'"
                  :aria-label="t('modelMarketplace.toolbar.multiplier', '倍率')"
                  :aria-pressed="showMultiplier"
                  @click="showMultiplier = !showMultiplier"
                >
                  <span
                    class="absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition"
                    :class="showMultiplier ? 'left-4' : 'left-0.5'"
                  ></span>
                </button>
              </div>

              <select v-model="sortMode" class="input h-9 w-36 text-sm">
                <option value="default">{{ t('modelMarketplace.sort.default', '默认排序') }}</option>
                <option value="price">{{ t('modelMarketplace.sort.price', '价格低到高') }}</option>
                <option value="channels">{{ t('modelMarketplace.sort.channels', '渠道数') }}</option>
                <option value="name">{{ t('modelMarketplace.sort.name', '模型名') }}</option>
              </select>

              <div class="inline-flex rounded-md border border-gray-200 bg-gray-50 p-1 dark:border-dark-700 dark:bg-dark-900/70">
                <button
                  type="button"
                  class="inline-flex h-7 items-center gap-1 rounded px-2 text-xs font-medium transition"
                  :class="viewMode === 'cards' ? 'bg-white text-cyan-700 shadow-sm dark:bg-dark-700 dark:text-cyan-200' : 'text-gray-500 dark:text-gray-400'"
                  @click="viewMode = 'cards'"
                >
                  <Icon name="grid" size="xs" />
                  {{ t('modelMarketplace.views.cards', '卡片') }}
                </button>
                <button
                  type="button"
                  class="inline-flex h-7 items-center gap-1 rounded px-2 text-xs font-medium transition"
                  :class="viewMode === 'table' ? 'bg-white text-cyan-700 shadow-sm dark:bg-dark-700 dark:text-cyan-200' : 'text-gray-500 dark:text-gray-400'"
                  @click="viewMode = 'table'"
                >
                  <Icon name="menu" size="xs" />
                  {{ t('modelMarketplace.views.table', '表格') }}
                </button>
              </div>

              <button
                type="button"
                data-testid="marketplace-token-unit-toggle"
                class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-gray-200 text-sm font-semibold text-gray-600 transition hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-700"
                :class="tokenPriceUnit === 'k' ? 'bg-cyan-50 text-cyan-700 dark:bg-cyan-950/40 dark:text-cyan-200' : ''"
                :title="t('modelMarketplace.toolbar.tokenUnit', 'Token 价格单位')"
                :aria-label="t('modelMarketplace.toolbar.tokenUnit', 'Token 价格单位')"
                @click="tokenPriceUnit = tokenPriceUnit === 'm' ? 'k' : 'm'"
              >
                {{ tokenPriceUnit.toUpperCase() }}
              </button>

              <button
                v-if="isAuthenticated"
                type="button"
                class="btn btn-secondary btn-sm"
                @click="calculatorOpen = !calculatorOpen"
              >
                <Icon name="calculator" size="sm" />
                {{ t('modelMarketplace.calculator.title') }}
              </button>
              <button
                @click="loadChannels"
                :disabled="loading"
                class="btn btn-secondary btn-sm"
                :title="t('common.refresh', 'Refresh')"
              >
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              </button>
            </div>
          </div>
        </section>

        <div class="grid gap-4 xl:grid-cols-[220px_minmax(0,1fr)]">
          <aside class="space-y-4 xl:sticky xl:top-20 xl:self-start">
            <section class="marketplace-panel rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="mb-2 flex items-center justify-between">
                <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('modelMarketplace.filters.title') }}
                </h2>
                <button
                  v-if="hasActiveFilters"
                  type="button"
                  class="text-xs font-medium text-cyan-700 hover:text-cyan-900 dark:text-cyan-300 dark:hover:text-cyan-200"
                  @click="clearAllFilters"
                >
                  {{ t('modelMarketplace.filters.clear') }}
                </button>
              </div>

              <div class="marketplace-filter-group">
                <p class="marketplace-filter-title">{{ t('modelMarketplace.filterGroups.providers', '供应商') }}</p>
                <button
                  type="button"
                  class="marketplace-filter-item"
                  :class="!selectedPlatform ? 'marketplace-filter-active' : ''"
                  @click="selectedPlatform = ''"
                >
                  <span>{{ t('modelMarketplace.filters.allPlatforms') }}</span>
                  <span>{{ modelRows.length }}</span>
                </button>
                <button
                  v-for="option in platformFilterOptions"
                  :key="option.platform"
                  type="button"
                  class="marketplace-filter-item"
                  :class="selectedPlatform === option.platform ? 'marketplace-filter-active' : ''"
                  @click="selectedPlatform = option.platform"
                >
                  <span class="inline-flex min-w-0 items-center gap-1.5">
                    <PlatformIcon :platform="option.platform as GroupPlatform" size="xs" />
                    <span class="truncate">{{ platformLabel(option.platform) }}</span>
                  </span>
                  <span>{{ option.count }}</span>
                </button>
              </div>

              <div class="marketplace-filter-group">
                <p class="marketplace-filter-title">{{ t('modelMarketplace.filterGroups.types', '模型类型') }}</p>
                <button
                  type="button"
                  class="marketplace-filter-item"
                  :class="!selectedModelType ? 'marketplace-filter-active' : ''"
                  @click="selectedModelType = ''"
                >
                  <span>{{ t('modelMarketplace.filterGroups.allTypes', '全部类型') }}</span>
                  <span>{{ modelRows.length }}</span>
                </button>
                <button
                  v-for="option in modelTypeOptions"
                  :key="option.type"
                  type="button"
                  class="marketplace-filter-item"
                  :class="selectedModelType === option.type ? 'marketplace-filter-active' : ''"
                  @click="selectedModelType = option.type"
                >
                  <span>{{ option.label }}</span>
                  <span>{{ option.count }}</span>
                </button>
              </div>

              <div class="marketplace-filter-group">
                <p class="marketplace-filter-title">{{ t('modelMarketplace.filterGroups.tags', '标签') }}</p>
                <button
                  type="button"
                  class="marketplace-filter-item"
                  :class="!selectedTag ? 'marketplace-filter-active' : ''"
                  @click="selectedTag = ''"
                >
                  <span>{{ t('modelMarketplace.filterGroups.allTags', '全部标签') }}</span>
                  <span>{{ modelRows.length }}</span>
                </button>
                <button
                  v-for="option in tagFilterOptions.slice(0, 8)"
                  :key="option.tag"
                  type="button"
                  class="marketplace-filter-item"
                  :class="selectedTag === option.tag ? 'marketplace-filter-active' : ''"
                  @click="selectedTag = option.tag"
                >
                  <span class="truncate">{{ option.tag }}</span>
                  <span>{{ option.count }}</span>
                </button>
              </div>

              <div class="marketplace-filter-group">
                <p class="marketplace-filter-title">{{ t('modelMarketplace.filterGroups.groups', '可用令牌分组') }}</p>
                <button
                  type="button"
                  class="marketplace-filter-item"
                  :class="!selectedGroupName ? 'marketplace-filter-active' : ''"
                  @click="selectedGroupName = ''"
                >
                  <span>{{ t('modelMarketplace.filterGroups.allGroups', '全部分组') }}</span>
                  <span>{{ t('common.all', '全部') }}</span>
                </button>
                <button
                  v-for="option in groupFilterOptions.slice(0, 8)"
                  :key="option.name"
                  type="button"
                  class="marketplace-filter-item"
                  :class="selectedGroupName === option.name ? 'marketplace-filter-active' : ''"
                  @click="selectedGroupName = option.name"
                >
                  <span class="truncate">{{ option.name }}</span>
                  <span>x{{ formatMultiplier(option.multiplier) }}</span>
                </button>
              </div>
            </section>
          </aside>

          <section class="min-w-0 space-y-4">
            <div class="marketplace-panel marketplace-results-bar rounded-lg border border-gray-200 bg-white px-4 py-3 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <span class="text-sm text-gray-500 dark:text-gray-400">
                  {{ t('modelMarketplace.filteredCount', { filtered: filteredRows.length, total: modelRows.length }) }}
                </span>
                <div class="flex flex-wrap gap-2 text-xs">
                  <span v-if="selectedPlatform" class="marketplace-active-chip">{{ platformLabel(selectedPlatform) }}</span>
                  <span v-if="selectedModelType" class="marketplace-active-chip">{{ modelTypeLabel(selectedModelType) }}</span>
                  <span v-if="selectedTag" class="marketplace-active-chip">{{ selectedTag }}</span>
                  <span v-if="selectedGroupName" class="marketplace-active-chip">{{ selectedGroupName }}</span>
                </div>
              </div>
            </div>

          <section
            v-if="isAuthenticated && calculatorOpen"
            class="marketplace-panel rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="mb-3 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('modelMarketplace.calculator.title') }}
              </h3>
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                @click="resetCalculatorInputs"
              >
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

            <div v-if="loading" class="grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
              <div
                v-for="index in 9"
                :key="index"
                class="h-52 animate-pulse rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
              ></div>
            </div>

            <div
              v-else-if="filteredRows.length === 0"
              class="marketplace-empty rounded-lg border border-gray-200 bg-white px-4 py-16 text-center shadow-sm dark:border-dark-700 dark:bg-dark-800"
            >
              <span class="marketplace-empty-icon">
                <Icon name="inbox" size="lg" />
              </span>
              <p class="mt-4 text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('modelMarketplace.empty') }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('modelMarketplace.emptyHint', '调整筛选条件或稍后刷新模型数据') }}
              </p>
            </div>

            <div
              v-else-if="viewMode === 'cards'"
              class="marketplace-card-grid grid gap-3 md:grid-cols-2 2xl:grid-cols-2"
            >
              <article
                v-for="row in filteredRows"
                :key="row.key"
                class="marketplace-model-card group cursor-pointer rounded-lg border border-gray-200 bg-white p-4 shadow-sm transition hover:border-cyan-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-800 dark:hover:border-cyan-700"
                @click="openModelDrawer(row)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="flex min-w-0 items-start gap-3">
                    <span
                      class="marketplace-model-avatar flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-sm font-bold text-white"
                      :class="modelAccentClass(row.platform)"
                    >
                      {{ modelInitial(row) }}
                    </span>
                    <div class="min-w-0">
                      <h3 class="truncate font-mono text-sm font-semibold text-gray-950 dark:text-white">
                        {{ row.name }}
                      </h3>
                      <p class="mt-1 flex min-w-0 items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
                        <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                        <span class="truncate">{{ platformLabel(row.platform) }}</span>
                      </p>
                    </div>
                  </div>
                  <span class="marketplace-type-pill shrink-0">{{ modelTypeLabel(modelTypeId(row)) }}</span>
                </div>

                <p class="mt-3 line-clamp-2 min-h-[2.5rem] text-sm leading-5 text-gray-600 dark:text-gray-300">
                  {{ modelDescription(row) }}
                </p>

                <div class="marketplace-card-price mt-4">
                  <div class="min-w-0">
                    <span class="block text-[11px] font-medium uppercase text-gray-400 dark:text-gray-500">
                      {{ t('modelMarketplace.columns.price') }}
                    </span>
                    <strong class="mt-1 block truncate text-sm font-semibold text-gray-950 dark:text-white">
                      {{ priceSummary(row) }}
                    </strong>
                  </div>
                  <span class="marketplace-billing-pill shrink-0">{{ billingSummary(row) }}</span>
                </div>

                <div class="mt-3 flex flex-wrap gap-1.5">
                  <span
                    v-for="tag in modelTags(row).slice(0, 3)"
                    :key="tag"
                    class="marketplace-tag"
                  >
                    {{ tag }}
                  </span>
                </div>

                <div class="mt-4 grid grid-cols-3 gap-0 overflow-hidden rounded-md border border-gray-100 bg-gray-50/70 text-xs dark:border-dark-700 dark:bg-dark-900/40">
                  <div class="marketplace-card-metric">
                    <span>{{ t('modelMarketplace.columns.channels') }}</span>
                    <strong>{{ row.channelNames.length }}</strong>
                  </div>
                  <div class="marketplace-card-metric">
                    <span>{{ t('modelMarketplace.columns.availability') }}</span>
                    <strong>{{ row.groupCount }}</strong>
                  </div>
                  <div class="marketplace-card-metric">
                    <span>{{ t('modelMarketplace.entryCount') }}</span>
                    <strong>{{ row.entries.length }}</strong>
                  </div>
                </div>

                <div class="mt-4 flex items-center gap-2 border-t border-gray-100 pt-3 dark:border-dark-700">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 text-gray-500 transition hover:bg-gray-50 dark:border-dark-700 dark:text-gray-300 dark:hover:bg-dark-700"
                    :title="t('common.copy')"
                    @click.stop="copyModelName(row)"
                  >
                    <Icon name="copy" size="sm" />
                  </button>
                  <a
                    href="/experience"
                    class="inline-flex h-8 min-w-0 flex-1 items-center justify-center gap-1.5 rounded-md border border-cyan-300/50 bg-cyan-50 px-3 text-xs font-semibold text-cyan-700 transition hover:bg-cyan-100 dark:border-cyan-700/50 dark:bg-cyan-950/40 dark:text-cyan-200 dark:hover:bg-cyan-900/50"
                    @click.stop="(e: MouseEvent) => { e.preventDefault(); router.push('/experience') }"
                  >
                    <Icon name="bolt" size="xs" />
                    {{ t('modelMarketplace.actions.tryNow', '立即体验') }}
                  </a>
                </div>
              </article>
            </div>

            <div v-else class="marketplace-panel overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <div class="overflow-x-auto">
                <table class="w-full min-w-[1120px] table-fixed text-sm">
                  <thead>
                    <tr class="border-b border-gray-100 bg-gray-50 text-xs font-semibold text-gray-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-gray-400">
                      <th class="w-10 px-4 py-3 text-left">
                        <span class="block h-4 w-4 rounded border border-gray-300 dark:border-dark-600"></span>
                      </th>
                      <th class="w-[250px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.model') }}</th>
                      <th class="w-[160px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.platform') }}</th>
                      <th class="px-4 py-3 text-left">{{ t('modelMarketplace.drawer.description', '描述') }}</th>
                      <th class="w-[220px] px-4 py-3 text-left">{{ t('modelMarketplace.filterGroups.tags', '标签') }}</th>
                      <th class="w-[150px] px-4 py-3 text-left">{{ t('modelMarketplace.drawer.billingType', '计费类型') }}</th>
                      <th class="w-[260px] px-4 py-3 text-left">{{ t('modelMarketplace.columns.price') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr
                      v-for="row in filteredRows"
                      :key="row.key"
                      class="cursor-pointer border-b border-gray-100 transition hover:bg-cyan-50/50 dark:border-dark-700 dark:hover:bg-cyan-950/20"
                      @click="openModelDrawer(row)"
                    >
                      <td class="px-4 py-4">
                        <span class="block h-4 w-4 rounded border border-gray-300 dark:border-dark-600"></span>
                      </td>
                      <td class="px-4 py-4">
                        <span class="marketplace-model-chip">
                          {{ row.name }}
                        </span>
                      </td>
                      <td class="px-4 py-4">
                        <span
                          :class="[
                            'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium',
                            platformBadgeClass(row.platform),
                          ]"
                        >
                          <PlatformIcon :platform="row.platform as GroupPlatform" size="xs" />
                          {{ platformLabel(row.platform) }}
                        </span>
                      </td>
                      <td class="px-4 py-4 text-gray-600 dark:text-gray-300">
                        <span class="line-clamp-1">{{ modelDescription(row) }}</span>
                      </td>
                      <td class="px-4 py-4">
                        <div class="flex flex-wrap gap-1">
                          <span v-for="tag in modelTags(row).slice(0, 3)" :key="tag" class="marketplace-tag">{{ tag }}</span>
                        </div>
                      </td>
                      <td class="px-4 py-4">
                        <span class="marketplace-billing-pill">{{ billingSummary(row) }}</span>
                      </td>
                      <td class="px-4 py-4 font-medium text-gray-900 dark:text-white">
                        {{ priceSummary(row) }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
      </div>
    </main>

    <Teleport to="body">
      <Transition name="marketplace-drawer">
        <div
          v-if="selectedRow"
          class="fixed inset-0 z-[90] bg-gray-950/55 backdrop-blur-sm"
          @click.self="closeModelDrawer"
        >
          <aside class="marketplace-drawer-panel fixed right-0 top-0 h-full w-full max-w-5xl overflow-y-auto bg-white shadow-2xl dark:bg-dark-900">
            <div class="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-gray-200 bg-white/95 px-5 py-4 backdrop-blur dark:border-dark-700 dark:bg-dark-900/95">
              <div class="flex min-w-0 items-start gap-3">
                <span
                  class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-sm font-bold text-white"
                  :class="modelAccentClass(selectedRow.platform)"
                >
                  {{ modelInitial(selectedRow) }}
                </span>
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h2 class="truncate font-mono text-lg font-bold text-gray-950 dark:text-white">
                      {{ selectedRow.name }}
                    </h2>
                    <button
                      type="button"
                      class="rounded-md p-1 text-cyan-600 transition hover:bg-cyan-50 dark:text-cyan-300 dark:hover:bg-cyan-950/40"
                      :title="t('common.copy')"
                      @click="copyModelName(selectedRow)"
                    >
                      <Icon name="copy" size="sm" />
                    </button>
                    <span class="marketplace-type-pill">{{ modelTypeLabel(modelTypeId(selectedRow)) }}</span>
                  </div>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ platformLabel(selectedRow.platform) }}</p>
                </div>
              </div>
              <button
                type="button"
                class="rounded-lg p-2 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                @click="closeModelDrawer"
              >
                <Icon name="x" size="md" />
              </button>
            </div>

            <div class="space-y-6 px-5 py-5">
              <section class="grid gap-5 lg:grid-cols-[1fr_480px]">
                <div>
                  <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('modelMarketplace.drawer.basicInfo', '基本信息') }}</h3>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.drawer.basicHint', '模型的详细描述和基本特性') }}</p>
                  <p class="mt-4 text-sm leading-7 text-gray-600 dark:text-gray-300">{{ modelDescription(selectedRow) }}</p>
                  <div class="mt-4 flex flex-wrap gap-2">
                    <span v-for="tag in modelTags(selectedRow)" :key="tag" class="marketplace-tag">{{ tag }}</span>
                  </div>
                </div>

                <div>
                  <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('modelMarketplace.drawer.apiEndpoints', 'API 端点') }}</h3>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.drawer.apiHint', '模型支持的接口端点信息') }}</p>
                  <div class="mt-4 space-y-3">
                    <div
                      v-for="endpoint in apiEndpointsForRow(selectedRow)"
                      :key="`${endpoint.method}-${endpoint.path}`"
                      class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700"
                    >
                      <span class="rounded border border-emerald-300 bg-emerald-50 px-2 py-1 text-[11px] font-bold text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300">
                        {{ endpoint.method }}
                      </span>
                      <div class="min-w-0 flex-1">
                        <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ endpoint.name }}</p>
                        <p class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ endpoint.path }}</p>
                      </div>
                      <button
                        type="button"
                        class="rounded-md p-1.5 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-700 dark:hover:text-gray-200"
                        :title="t('common.copy')"
                        @click="copyToClipboard(endpoint.path)"
                      >
                        <Icon name="copy" size="sm" />
                      </button>
                    </div>
                  </div>
                </div>
              </section>

              <section class="border-t border-gray-200 pt-5 dark:border-dark-700">
                <div class="flex flex-wrap items-end justify-between gap-3">
                  <div>
                    <h3 class="text-sm font-semibold text-gray-950 dark:text-white">{{ t('modelMarketplace.drawer.groupPricing', '分组价格') }}</h3>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('modelMarketplace.drawer.groupPricingHint', '不同用户分组的价格信息') }}</p>
                  </div>
                  <div class="flex flex-wrap items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                    <span v-for="(name, index) in drawerGroupNames(selectedRow).slice(0, 6)" :key="name" class="inline-flex items-center gap-1">
                      <span v-if="index > 0">→</span>
                      <span class="rounded-full border border-gray-200 px-2 py-1 dark:border-dark-700">{{ name }}</span>
                    </span>
                  </div>
                </div>

                <div class="mt-4 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-700">
                  <table class="w-full min-w-[760px] text-sm">
                    <thead>
                      <tr class="border-b border-gray-100 bg-gray-50 text-xs font-semibold text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400">
                        <th class="px-4 py-3 text-left">{{ t('modelMarketplace.calculator.group') }}</th>
                        <th class="px-4 py-3 text-left">{{ t('modelMarketplace.drawer.channel', '渠道') }}</th>
                        <th class="px-4 py-3 text-left">{{ t('modelMarketplace.drawer.billingType', '计费类型') }}</th>
                        <th class="px-4 py-3 text-left">{{ t('modelMarketplace.drawer.priceSummary', '价格摘要') }}</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr
                        v-for="item in drawerPriceRows(selectedRow)"
                        :key="`${item.channelName}-${item.group.id}`"
                        class="border-b border-gray-100 last:border-0 dark:border-dark-700"
                      >
                        <td class="px-4 py-4">
                          <div class="flex flex-wrap items-center gap-2">
                            <GroupBadge
                              :name="item.group.name"
                              :platform="item.group.platform as GroupPlatform"
                              :subscription-type="(item.group.subscription_type || 'standard') as SubscriptionType"
                              :rate-multiplier="item.group.rate_multiplier"
                              :user-rate-multiplier="userGroupRates[item.group.id] ?? null"
                              :show-rate="showMultiplier"
                              :always-show-rate="showMultiplier"
                            />
                          </div>
                        </td>
                        <td class="px-4 py-4 text-gray-600 dark:text-gray-300">{{ item.channelName }}</td>
                        <td class="px-4 py-4">
                          <span class="marketplace-billing-pill">{{ billingModeLabel(item.billingMode) }}</span>
                        </td>
                        <td class="px-4 py-4 font-medium text-orange-600 dark:text-orange-300">
                          {{ item.price }}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </section>
            </div>
          </aside>
        </div>
      </Transition>
    </Teleport>
  </component>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicHeader from '@/components/layout/PublicHeader.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import userChannelsAPI, {
  type UserAvailableChannel,
  type UserAvailableGroup,
  type UserSupportedModel,
  type UserSupportedModelPricing,
} from '@/api/channels'
import userGroupsAPI from '@/api/groups'
import { BILLING_MODE_IMAGE, BILLING_MODE_PER_REQUEST, BILLING_MODE_TOKEN, type BillingMode } from '@/constants/channel'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
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

interface MarketplaceEndpoint {
  method: 'POST' | 'GET'
  name: string
  path: string
}

interface DrawerPriceRow {
  channelName: string
  group: UserAvailableGroup
  billingMode: BillingMode
  price: string
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const channels = ref<UserAvailableChannel[]>([])
const userGroupRates = ref<Record<number, number>>({})
const loading = ref(false)
const searchQuery = ref('')
const selectedPlatform = ref('')
const selectedModelType = ref('')
const selectedGroupName = ref('')
const selectedTag = ref('')
const sortMode = ref<'default' | 'price' | 'channels' | 'name'>('default')
const viewMode = ref<'cards' | 'table'>('cards')
const tokenPriceUnit = ref<'m' | 'k'>('m')
const showMultiplier = ref(false)
const selectedRow = ref<ModelMarketplaceRow | null>(null)
const calculatorOpen = ref(false)
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
const isAuthenticated = computed(() => authStore.isAuthenticated)
const currentPath = computed(() => '/models')
const pageShell = computed(() => (isAuthenticated.value ? AppLayout : 'div'))
const publicShellClass = computed(() => (
  isAuthenticated.value
    ? ''
    : 'home-shell min-h-screen overflow-hidden text-slate-100'
))
const mainShellClass = computed(() => (isAuthenticated.value ? '' : 'relative'))
const marketplaceContainerClass = computed(() => (
  isAuthenticated.value
    ? 'w-full max-w-none space-y-5'
    : 'w-full max-w-none space-y-5 px-0 py-6 lg:py-8'
))
const hasActiveFilters = computed(() =>
  Boolean(selectedPlatform.value || selectedModelType.value || selectedGroupName.value || selectedTag.value || searchQuery.value.trim()),
)

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

const platformFilterOptions = computed(() =>
  platformOptions.value.map((platform) => ({
    platform,
    count: modelRows.value.filter((row) => row.platform === platform).length,
  })),
)

const modelTypeOptions = computed(() => {
  const byType = new Map<string, number>()
  for (const row of modelRows.value) {
    const type = modelTypeId(row)
    byType.set(type, (byType.get(type) ?? 0) + 1)
  }
  return Array.from(byType.entries()).map(([type, count]) => ({
    type,
    label: modelTypeLabel(type),
    count,
  }))
})

const groupFilterOptions = computed(() => {
  const byGroup = new Map<string, { name: string; multiplier: number; count: number }>()
  for (const row of modelRows.value) {
    const seen = new Set<string>()
    for (const entry of row.entries) {
      for (const group of entry.groups) {
        if (seen.has(group.name)) continue
        seen.add(group.name)
        const current = byGroup.get(group.name)
        byGroup.set(group.name, {
          name: group.name,
          multiplier: userGroupRates.value[group.id] ?? group.rate_multiplier,
          count: (current?.count ?? 0) + 1,
        })
      }
    }
  }
  return Array.from(byGroup.values()).sort((a, b) => b.count - a.count || a.name.localeCompare(b.name))
})

const tagFilterOptions = computed(() => {
  const byTag = new Map<string, number>()
  for (const row of modelRows.value) {
    for (const tag of modelTags(row)) {
      byTag.set(tag, (byTag.get(tag) ?? 0) + 1)
    }
  }
  return Array.from(byTag.entries())
    .map(([tag, count]) => ({ tag, count }))
    .sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag))
})

const filteredRows = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  const rows = modelRows.value.filter((row) => {
    if (selectedPlatform.value && row.platform !== selectedPlatform.value) return false
    if (selectedModelType.value && modelTypeId(row) !== selectedModelType.value) return false
    if (selectedGroupName.value && !row.entries.some((entry) => entry.groups.some((group) => group.name === selectedGroupName.value))) return false
    if (selectedTag.value && !modelTags(row).includes(selectedTag.value)) return false
    if (!q) return true
    return (
      row.name.toLowerCase().includes(q) ||
      row.platform.toLowerCase().includes(q) ||
      platformLabel(row.platform).toLowerCase().includes(q) ||
      modelTypeLabel(modelTypeId(row)).toLowerCase().includes(q) ||
      modelTags(row).some((tag) => tag.toLowerCase().includes(q)) ||
      row.entries.some(
        (entry) =>
          entry.channelName.toLowerCase().includes(q) ||
          entry.channelDescription.toLowerCase().includes(q) ||
          entry.groups.some((group) => group.name.toLowerCase().includes(q)),
      )
    )
  })

  return [...rows].sort((a, b) => {
    if (sortMode.value === 'name') return a.name.localeCompare(b.name)
    if (sortMode.value === 'channels') return b.channelNames.length - a.channelNames.length || a.name.localeCompare(b.name)
    if (sortMode.value === 'price') return comparablePrice(a) - comparablePrice(b) || a.name.localeCompare(b.name)
    return 0
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

function modelInitial(row: ModelMarketplaceRow): string {
  const initial = (row.name || row.platform || 'AI').trim().slice(0, 1).toUpperCase()
  return initial || 'A'
}

function modelAccentClass(platform: string): string {
  const value = platform.toLowerCase()
  if (value.includes('anthropic') || value.includes('claude')) return 'bg-amber-500'
  if (value.includes('openai') || value.includes('gpt')) return 'bg-emerald-500'
  if (value.includes('google') || value.includes('gemini')) return 'bg-blue-500'
  if (value.includes('xai') || value.includes('grok')) return 'bg-fuchsia-500'
  return 'bg-cyan-500'
}

function modelTypeId(row: ModelMarketplaceRow): string {
  const name = row.name.toLowerCase()
  const billingModes = new Set(row.entries.map((entry) => entry.model.pricing?.billing_mode).filter(Boolean))
  if (billingModes.has(BILLING_MODE_IMAGE) || name.includes('image') || name.includes('dall') || name.includes('pixverse')) return 'image'
  if (name.includes('audio') || name.includes('speech') || name.includes('tts') || name.includes('voice')) return 'audio'
  if (name.includes('video') || name.includes('lipsync')) return 'video'
  if (name.includes('embed') || name.includes('rerank')) return 'search'
  if (name.includes('moderation') || name.includes('classifier')) return 'text'
  return 'chat'
}

function modelTypeLabel(type: string): string {
  const labels: Record<string, string> = {
    chat: t('modelMarketplace.types.chat', '对话'),
    search: t('modelMarketplace.types.search', '检索'),
    image: t('modelMarketplace.types.image', '图像'),
    text: t('modelMarketplace.types.text', '文本'),
    audio: t('modelMarketplace.types.audio', '音视频'),
    video: t('modelMarketplace.types.video', '视频'),
  }
  return labels[type] || type
}

function modelDescription(row: ModelMarketplaceRow): string {
  const descriptions = row.entries
    .map((entry) => entry.channelDescription)
    .filter((description) => description.trim().length > 0)
  if (descriptions.length > 0) return descriptions[0]
  return t('modelMarketplace.drawer.generatedDescription', {
    model: row.name,
    platform: platformLabel(row.platform),
  }, `${row.name} is available through ${platformLabel(row.platform)} channels with grouped pricing and endpoint metadata.`)
}

function modelTags(row: ModelMarketplaceRow): string[] {
  const tags = new Set<string>()
  const type = modelTypeId(row)
  tags.add(modelTypeLabel(type))
  if (row.entries.some((entry) => entry.model.pricing?.billing_mode === BILLING_MODE_TOKEN)) tags.add(t('availableChannels.pricing.billingModeToken'))
  if (row.entries.some((entry) => entry.model.pricing?.billing_mode === BILLING_MODE_PER_REQUEST)) tags.add(t('availableChannels.pricing.billingModePerRequest'))
  if (row.entries.some((entry) => entry.model.pricing?.billing_mode === BILLING_MODE_IMAGE)) tags.add(t('availableChannels.pricing.billingModeImage'))
  if (row.groupCount > 0) tags.add(t('modelMarketplace.availability.groups', { count: row.groupCount }))
  return Array.from(tags)
}

function comparablePrice(row: ModelMarketplaceRow): number {
  const values = row.entries
    .map((entry) => {
      const pricing = entry.model.pricing
      if (!pricing) return null
      const price = pricing.input_price ?? pricing.per_request_price ?? pricing.image_output_price ?? pricing.output_price
      return price == null ? null : price * entryPriceMultiplier(entry)
    })
    .filter((value): value is number => value !== null)
  return values.length > 0 ? Math.min(...values) : Number.MAX_SAFE_INTEGER
}

function formatMultiplier(value: number): string {
  if (!Number.isFinite(value)) return '1'
  return value.toFixed(2).replace(/\.?0+$/, '')
}

function clearAllFilters() {
  searchQuery.value = ''
  selectedPlatform.value = ''
  selectedModelType.value = ''
  selectedGroupName.value = ''
  selectedTag.value = ''
}

function openModelDrawer(row: ModelMarketplaceRow) {
  selectedRow.value = row
}

function closeModelDrawer() {
  selectedRow.value = null
}

async function copyModelName(row: ModelMarketplaceRow) {
  await copyToClipboard(row.name)
}

function apiEndpointsForRow(row: ModelMarketplaceRow): MarketplaceEndpoint[] {
  const type = modelTypeId(row)
  if (type === 'image') {
    return [
      { method: 'POST', name: t('modelMarketplace.endpoints.imageEdit', '图像编辑'), path: '/v1/images/edits' },
      { method: 'POST', name: t('modelMarketplace.endpoints.imageGeneration', '图像生成'), path: '/v1/images/generations' },
    ]
  }
  if (type === 'audio') {
    return [
      { method: 'POST', name: t('modelMarketplace.endpoints.speech', '语音生成'), path: '/v1/audio/speech' },
      { method: 'POST', name: t('modelMarketplace.endpoints.transcription', '语音转写'), path: '/v1/audio/transcriptions' },
    ]
  }
  if (type === 'video') {
    return [
      { method: 'POST', name: t('modelMarketplace.endpoints.videoGeneration', '视频生成'), path: '/v1/videos/generations' },
    ]
  }
  if (type === 'search') {
    return [
      { method: 'POST', name: t('modelMarketplace.endpoints.embeddings', '向量嵌入'), path: '/v1/embeddings' },
      { method: 'POST', name: t('modelMarketplace.endpoints.rerank', '重排'), path: '/v1/rerank' },
    ]
  }
  return [
    { method: 'POST', name: t('modelMarketplace.endpoints.chatCompletions', '聊天补全'), path: '/v1/chat/completions' },
    { method: 'POST', name: t('modelMarketplace.endpoints.responses', 'Responses'), path: '/v1/responses' },
  ]
}

function drawerGroupNames(row: ModelMarketplaceRow): string[] {
  const names = new Set<string>()
  for (const entry of row.entries) {
    for (const group of entry.groups) names.add(group.name)
  }
  return Array.from(names)
}

function drawerPriceRows(row: ModelMarketplaceRow): DrawerPriceRow[] {
  const rows: DrawerPriceRow[] = []
  for (const entry of row.entries) {
    for (const group of entry.groups) {
      const pricing = entry.model.pricing
      rows.push({
        channelName: entry.channelName,
        group,
        billingMode: pricing?.billing_mode ?? BILLING_MODE_TOKEN,
        price: pricing ? priceSummaryForPricing(pricing, group) : t('availableChannels.noPricing'),
      })
    }
  }
  return rows
}

function priceSummaryForPricing(pricing: UserSupportedModelPricing, group: UserAvailableGroup): string {
  const multiplier = showMultiplier.value ? calculatorGroupMultiplier(group) : 1
  if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    const input = pricing.input_price == null ? null : pricing.input_price * multiplier
    const output = pricing.output_price == null ? null : pricing.output_price * multiplier
    return t('modelMarketplace.pricing.tokenSummary', {
      input: formatScaled(input, tokenPriceScale.value),
      output: formatScaled(output, tokenPriceScale.value),
    })
  }
  if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    const price = (pricing.image_output_price ?? pricing.per_request_price)
    return t('modelMarketplace.pricing.imageSummary', { price: formatScaled(price == null ? null : price * multiplier, 1) })
  }
  const price = pricing.per_request_price
  return t('modelMarketplace.pricing.requestSummary', { price: formatScaled(price == null ? null : price * multiplier, 1) })
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
  const pricedEntries = row.entries.filter(
    (entry): entry is ModelMarketplaceEntry & { model: UserSupportedModel & { pricing: UserSupportedModelPricing } } => entry.model.pricing !== null,
  )
  if (pricedEntries.length === 0) return t('availableChannels.noPricing')

  const tokenEntries = pricedEntries.filter((entry) => entry.model.pricing.billing_mode === BILLING_MODE_TOKEN)
  if (tokenEntries.length > 0) {
    const input = minNumber(tokenEntries.map((entry) => multiplyPrice(entry.model.pricing.input_price, entryPriceMultiplier(entry))))
    const output = minNumber(tokenEntries.map((entry) => multiplyPrice(entry.model.pricing.output_price, entryPriceMultiplier(entry))))
    return t('modelMarketplace.pricing.tokenSummary', {
      input: formatScaled(input, tokenPriceScale.value),
      output: formatScaled(output, tokenPriceScale.value),
    })
  }

  const requestEntries = pricedEntries.filter((entry) => entry.model.pricing.billing_mode === BILLING_MODE_PER_REQUEST)
  if (requestEntries.length > 0) {
    const price = minNumber(requestEntries.map((entry) => multiplyPrice(entry.model.pricing.per_request_price, entryPriceMultiplier(entry))))
    return t('modelMarketplace.pricing.requestSummary', { price: formatScaled(price, 1) })
  }

  const imageEntries = pricedEntries.filter((entry) => entry.model.pricing.billing_mode === BILLING_MODE_IMAGE)
  if (imageEntries.length > 0) {
    const price = minNumber(imageEntries.map((entry) => {
      const pricing = entry.model.pricing
      return multiplyPrice(pricing.image_output_price ?? pricing.per_request_price, entryPriceMultiplier(entry))
    }))
    return t('modelMarketplace.pricing.imageSummary', { price: formatScaled(price, 1) })
  }

  return t('availableChannels.noPricing')
}

const tokenPriceScale = computed(() => (tokenPriceUnit.value === 'm' ? 1_000_000 : 1_000))

function entryPriceMultiplier(entry: ModelMarketplaceEntry): number {
  if (!showMultiplier.value || entry.groups.length === 0) return 1
  return Math.min(...entry.groups.map(calculatorGroupMultiplier))
}

function multiplyPrice(price: number | null, multiplier: number): number | null {
  return price == null ? null : price * multiplier
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
      isAuthenticated.value ? userChannelsAPI.getAvailable() : userChannelsAPI.getPublicMarketplace(),
      isAuthenticated.value
        ? userGroupsAPI.getUserGroupRates().catch((err: unknown) => {
            console.error('Failed to load user group rates:', err)
            return {} as Record<number, number>
          })
        : Promise.resolve({} as Record<number, number>),
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
watch(isAuthenticated, loadChannels)

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  loadChannels()
})
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

.home-shell > main {
  position: relative;
  z-index: 1;
  width: 100%;
}

.home-shell .marketplace-hero,
.home-shell .marketplace-toolbar,
.home-shell .marketplace-results-bar,
.home-shell aside .marketplace-panel,
.home-shell .marketplace-empty,
.home-shell .marketplace-card-grid,
.home-shell .marketplace-panel:has(table) {
  border-radius: 0.75rem;
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

.marketplace-public-nav-link {
  position: relative;
}

.marketplace-public-nav-active {
  background: rgba(8, 145, 178, 0.1);
  color: rgb(103 232 249);
}

.marketplace-public-nav-active::after {
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

.home-light {
  color: rgb(51 65 85);
  background:
    radial-gradient(circle at 14% 5%, rgba(8, 145, 178, 0.12), transparent 31rem),
    radial-gradient(circle at 86% 28%, rgba(56, 189, 248, 0.1), transparent 30rem),
    linear-gradient(180deg, #f8fafc, #eef6fb 46rem, #ffffff);
}

.home-light .home-grid {
  background-image:
    linear-gradient(rgba(15, 23, 42, 0.065) 1px, transparent 1px),
    linear-gradient(90deg, rgba(15, 23, 42, 0.065) 1px, transparent 1px);
  opacity: 0.65;
}

.home-light header {
  border-bottom-color: rgba(15, 23, 42, 0.08);
  background: rgba(255, 255, 255, 0.88);
}

.home-light h1,
.home-light h2,
.home-light h3,
.home-light .text-white {
  color: rgb(15 23 42);
}

.home-light .text-slate-500 {
  color: rgb(100 116 139);
}

.home-light .text-slate-400 {
  color: rgb(71 85 105);
}

.home-light .text-slate-300,
.home-light .text-slate-200 {
  color: rgb(51 65 85);
}

.home-light .text-cyan-100 {
  color: rgb(14 116 144);
}

.home-light .border-white\/10,
.home-light .border-cyan-300\/25,
.home-light .border-cyan-300\/30 {
  border-color: rgba(8, 145, 178, 0.22);
}

.home-light .bg-white\/8 {
  background-color: rgba(255, 255, 255, 0.7);
}

.home-light .bg-cyan-300\/10 {
  background-color: rgba(8, 145, 178, 0.08);
}

.home-light .home-icon-button {
  color: rgb(71 85 105);
}

.home-light .home-icon-button:hover {
  background: rgba(15, 23, 42, 0.06);
  color: rgb(15 23 42);
}

.home-light .marketplace-public-nav-active {
  border: 1px solid rgba(8, 145, 178, 0.18);
  background: rgba(240, 249, 255, 0.9);
  color: rgb(14 116 144);
  box-shadow: 0 6px 18px rgba(8, 145, 178, 0.08);
}

.home-shell .marketplace-hero h1 {
  color: white;
}

.home-shell .marketplace-hero p {
  color: rgb(148 163 184);
}

.home-shell .marketplace-panel,
.home-shell .marketplace-model-card,
.home-shell .marketplace-empty {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.68);
  box-shadow: 0 14px 34px rgba(2, 6, 23, 0.2);
  backdrop-filter: blur(18px);
}

.home-shell .marketplace-hero {
  background:
    linear-gradient(135deg, rgba(15, 23, 42, 0.74), rgba(15, 23, 42, 0.52)),
    radial-gradient(circle at 84% 14%, rgba(103, 232, 249, 0.14), transparent 22rem);
}

.home-shell .marketplace-model-card:hover {
  border-color: rgba(103, 232, 249, 0.42);
  box-shadow: 0 22px 60px rgba(8, 145, 178, 0.18);
}

.home-shell .marketplace-panel select,
.home-shell .marketplace-panel input {
  border-color: rgba(148, 163, 184, 0.24);
  background-color: rgba(15, 23, 42, 0.72);
  color: rgb(226 232 240);
}

.home-light .marketplace-hero h1 {
  color: rgb(15 23 42);
}

.home-light .marketplace-hero p {
  color: rgb(71 85 105);
}

.home-light .marketplace-panel,
.home-light .marketplace-model-card,
.home-light .marketplace-empty {
  border-color: rgba(8, 145, 178, 0.16);
  background: rgba(255, 255, 255, 0.8);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.06);
  backdrop-filter: blur(18px);
}

.home-light .marketplace-hero {
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.9), rgba(240, 249, 255, 0.74)),
    radial-gradient(circle at 86% 18%, rgba(8, 145, 178, 0.12), transparent 22rem);
}

.home-light .marketplace-panel select,
.home-light .marketplace-panel input {
  border-color: rgba(8, 145, 178, 0.2);
  background-color: rgba(255, 255, 255, 0.82);
  color: rgb(15 23 42);
}

.marketplace-stat,
.marketplace-active-chip,
.marketplace-tag,
.marketplace-billing-pill,
.marketplace-type-pill,
.marketplace-model-chip {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  border: 1px solid rgb(226 232 240);
  background: rgb(248 250 252);
  color: rgb(71 85 105);
}

.marketplace-stat {
  justify-content: center;
  min-height: 2.25rem;
  padding: 0.55rem 0.75rem;
  font-weight: 600;
}

.marketplace-active-chip,
.marketplace-tag,
.marketplace-billing-pill,
.marketplace-type-pill,
.marketplace-model-chip {
  padding: 0.18rem 0.55rem;
  font-size: 0.75rem;
  line-height: 1rem;
}

.marketplace-type-pill {
  background: rgb(239 246 255);
  color: rgb(37 99 235);
}

.marketplace-billing-pill {
  background: rgb(245 243 255);
  color: rgb(109 40 217);
}

.marketplace-tag {
  background: rgb(236 253 245);
  color: rgb(4 120 87);
}

.marketplace-model-chip {
  max-width: 100%;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-weight: 600;
  color: rgb(15 23 42);
}

.marketplace-filter-group {
  margin-top: 1rem;
}

.marketplace-filter-group:first-of-type {
  margin-top: 0;
}

.marketplace-filter-title {
  margin-bottom: 0.45rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: rgb(100 116 139);
}

.marketplace-filter-item {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  border: 1px solid transparent;
  border-radius: 0.375rem;
  min-height: 2rem;
  padding: 0.42rem 0.55rem;
  text-align: left;
  font-size: 0.8125rem;
  color: rgb(51 65 85);
  transition: background-color 160ms ease, color 160ms ease;
}

.marketplace-filter-item:hover {
  background: rgb(248 250 252);
}

.marketplace-filter-active {
  border-color: rgba(8, 145, 178, 0.18);
  background: rgba(240, 249, 255, 0.82);
  color: rgb(14 116 144);
  font-weight: 600;
  box-shadow: inset 3px 0 0 rgba(8, 145, 178, 0.38);
}

.marketplace-card-metric {
  min-width: 0;
  border-right: 1px solid rgb(226 232 240);
  padding: 0.55rem 0.65rem;
}

.marketplace-card-metric:last-child {
  border-right: 0;
}

.marketplace-card-metric span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: rgb(100 116 139);
}

.marketplace-card-metric strong {
  margin-top: 0.25rem;
  display: block;
  color: rgb(15 23 42);
}

.marketplace-hero {
  overflow: hidden;
}

.marketplace-toolbar,
.marketplace-results-bar {
  box-shadow: 0 8px 22px rgba(15, 23, 42, 0.045);
}

.marketplace-toolbar select.input {
  height: 2.25rem;
  padding-top: 0;
  padding-bottom: 0;
  line-height: 2.125rem;
}

.marketplace-model-card {
  min-height: 18.5rem;
}

.marketplace-model-card:hover {
  transform: translateY(-1px);
}

.marketplace-model-avatar {
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.22);
}

.marketplace-card-price {
  display: flex;
  min-height: 3.35rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid rgb(226 232 240);
  background: linear-gradient(180deg, rgb(248 250 252), rgb(255 255 255));
  padding: 0.65rem 0.75rem;
}

.marketplace-empty {
  min-height: 12rem;
}

.marketplace-empty-icon {
  margin-inline: auto;
  display: inline-flex;
  height: 3rem;
  width: 3rem;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  border: 1px solid rgb(226 232 240);
  background: rgb(248 250 252);
  color: rgb(100 116 139);
}

.marketplace-drawer-enter-active,
.marketplace-drawer-leave-active {
  transition: opacity 180ms ease;
}

.marketplace-drawer-enter-active .marketplace-drawer-panel,
.marketplace-drawer-leave-active .marketplace-drawer-panel {
  transition: transform 220ms ease;
}

.marketplace-drawer-enter-from,
.marketplace-drawer-leave-to {
  opacity: 0;
}

.marketplace-drawer-enter-from .marketplace-drawer-panel,
.marketplace-drawer-leave-to .marketplace-drawer-panel {
  transform: translateX(100%);
}

.dark .marketplace-stat,
.dark .marketplace-active-chip,
.dark .marketplace-tag,
.dark .marketplace-billing-pill,
.dark .marketplace-type-pill,
.dark .marketplace-model-chip {
  border-color: rgb(51 65 85);
  background: rgba(15, 23, 42, 0.72);
  color: rgb(203 213 225);
}

.dark .marketplace-type-pill {
  color: rgb(147 197 253);
}

.dark .marketplace-billing-pill {
  color: rgb(196 181 253);
}

.dark .marketplace-tag {
  color: rgb(110 231 183);
}

.dark .marketplace-filter-title {
  color: rgb(148 163 184);
}

.dark .marketplace-filter-item {
  color: rgb(203 213 225);
}

.dark .marketplace-filter-item:hover {
  background: rgba(51, 65, 85, 0.55);
}

.dark .marketplace-filter-active {
  border-color: rgba(103, 232, 249, 0.22);
  background: rgba(8, 145, 178, 0.14);
  color: rgb(165 243 252);
  box-shadow: inset 3px 0 0 rgba(103, 232, 249, 0.36);
}

.dark .marketplace-card-metric {
  border-right-color: rgb(51 65 85);
}

.dark .marketplace-card-metric span {
  color: rgb(148 163 184);
}

.dark .marketplace-card-metric strong {
  color: white;
}

.home-shell .marketplace-stat,
.home-shell .marketplace-active-chip,
.home-shell .marketplace-tag,
.home-shell .marketplace-billing-pill,
.home-shell .marketplace-type-pill,
.home-shell .marketplace-model-chip {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.55);
  color: rgb(203 213 225);
}

.home-shell .marketplace-filter-item {
  color: rgb(203 213 225);
}

.home-shell .marketplace-filter-title {
  color: rgb(148 163 184);
}

.home-shell .marketplace-filter-item:hover {
  background: rgba(255, 255, 255, 0.06);
}

.home-shell .marketplace-filter-active {
  border-color: rgba(103, 232, 249, 0.22);
  background: rgba(8, 145, 178, 0.14);
  color: rgb(165 243 252);
  box-shadow: inset 3px 0 0 rgba(103, 232, 249, 0.34);
}

.home-shell .marketplace-card-price,
.home-shell .marketplace-empty-icon {
  border-color: rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.46);
}

.home-shell .marketplace-card-metric {
  border-right-color: rgba(148, 163, 184, 0.16);
}

.home-light .marketplace-stat,
.home-light .marketplace-active-chip,
.home-light .marketplace-tag,
.home-light .marketplace-billing-pill,
.home-light .marketplace-type-pill,
.home-light .marketplace-model-chip {
  border-color: rgba(8, 145, 178, 0.16);
  background: rgba(255, 255, 255, 0.72);
  color: rgb(51 65 85);
}

.home-light .marketplace-filter-item {
  color: rgb(51 65 85);
}

.home-light .marketplace-filter-title {
  color: rgb(71 85 105);
}

.home-light .marketplace-filter-item:hover {
  background: rgba(15, 23, 42, 0.05);
}

.home-light .marketplace-filter-active {
  border-color: rgba(8, 145, 178, 0.18);
  background: rgba(255, 255, 255, 0.78);
  color: rgb(14 116 144);
  box-shadow: inset 3px 0 0 rgba(8, 145, 178, 0.32);
}

.home-light .marketplace-card-price,
.home-light .marketplace-empty-icon {
  border-color: rgba(8, 145, 178, 0.14);
  background: rgba(255, 255, 255, 0.66);
}

@media (max-width: 640px) {
  .marketplace-stats-grid {
    min-width: 0;
    width: 100%;
  }

  .marketplace-model-card {
    min-height: auto;
  }
}

/* 移动端菜单动画 */
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
</style>
