<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { EnduroStoredCollection } from '~/openapi-generator'

const columns: TableColumn<EnduroStoredCollection>[] = [
  {
    accessorKey: 'id',
    header: 'ID'
  },
  {
    accessorKey: 'name',
    header: 'Name'
  },
  {
    accessorKey: 'startedAt',
    header: 'Started'
  },
  {
    accessorKey: 'completedAt',
    header: 'Stored'
  },
  {
    accessorKey: 'status',
    header: 'Status'
  }
]

const {
  statusOptions,
  dateOptions,
  fieldOptions,
  selectedStatus,
  selectedDate,
  selectedField,
  query,
  isLoading,
  hasError,
  validQuery,
  rows,
  queryHelp,
  queryError,
  hasRows,
  canGoPrev,
  canGoNext,
  showPager,
  statusColor,
  formatDateTime,
  onSubmit,
  onReset,
  onRetry,
  onGoHome,
  onGoPrev,
  onGoNext,
  onCollectionRowSelect
} = useCollectionsBrowser()

const {
  collectionsSearchOpen,
  setCollectionsSearchOpen
} = useDashboardUiOptions()

const hasActiveSearch = computed(() => (
  selectedStatus.value !== 'all'
  || selectedDate.value !== 'all'
  || selectedField.value !== 'name'
  || query.value.trim().length > 0
))

const searchButtonLabel = computed(() => (
  hasActiveSearch.value && !collectionsSearchOpen.value ? 'Search active' : 'Search'
))

function onToggleSearch() {
  setCollectionsSearchOpen(!collectionsSearchOpen.value)
}

onMounted(() => {
  if (hasActiveSearch.value) {
    setCollectionsSearchOpen(true, { persist: false })
  }
})

watch(hasActiveSearch, (active) => {
  if (active && !collectionsSearchOpen.value) {
    setCollectionsSearchOpen(true, { persist: false })
  }
})
</script>

<script lang="ts">
export { useCollectionsListData } from '~/loaders/collections-list'
</script>

<template>
  <AppPageContainer>
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-xl font-semibold text-highlighted">
          Collections
        </h1>
      </div>

      <div
        v-if="!hasError"
        class="flex flex-wrap gap-1.5"
      >
        <UButton
          label="Batch import"
          icon="i-lucide-folder-input"
          color="neutral"
          variant="outline"
          size="sm"
          to="/collections/batch"
        />
        <UButton
          label="Bulk operation"
          icon="i-lucide-list-checks"
          color="neutral"
          variant="outline"
          size="sm"
          to="/collections/bulk"
          :disabled="!hasRows"
        />
        <UButton
          :label="searchButtonLabel"
          icon="i-lucide-search"
          :trailing-icon="collectionsSearchOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
          color="neutral"
          :variant="collectionsSearchOpen ? 'soft' : 'outline'"
          size="sm"
          aria-controls="collections-search-panel"
          :aria-expanded="collectionsSearchOpen"
          :aria-pressed="collectionsSearchOpen"
          @click="onToggleSearch"
        />
      </div>
    </div>

    <UAlert
      v-if="hasError"
      color="warning"
      variant="subtle"
      title="Search error"
      description="Could not connect to the API server. Try again in a few seconds."
    >
      <template #actions>
        <UButton
          label="Retry"
          size="sm"
          color="warning"
          variant="outline"
          :loading="isLoading"
          @click="onRetry"
        />
      </template>
    </UAlert>

    <UCard
      v-if="!hasError && collectionsSearchOpen"
      id="collections-search-panel"
      :ui="{ header: 'p-3 sm:px-3 sm:py-2.5', body: 'p-3 sm:p-3' }"
    >
      <template #header>
        <h2 class="text-sm font-semibold text-highlighted">
          Search
        </h2>
      </template>

      <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-12 gap-2">
        <div class="sm:col-span-1 md:col-span-2">
          <UFormField label="Status">
            <USelect
              v-model="selectedStatus"
              :items="statusOptions"
              size="sm"
              class="w-full"
            />
          </UFormField>
        </div>
        <div class="sm:col-span-1 md:col-span-2">
          <UFormField label="Created">
            <USelect
              v-model="selectedDate"
              :items="dateOptions"
              size="sm"
              class="w-full"
            />
          </UFormField>
        </div>
        <div class="sm:col-span-2 md:col-span-5 lg:col-span-6">
          <UFormField
            label="Search query"
            :error="queryError || undefined"
          >
            <UFieldGroup
              size="sm"
              class="w-full"
            >
              <UInput
                v-model="query"
                placeholder="Search"
                autofocus
                size="sm"
                class="w-full"
                :color="validQuery === false ? 'error' : 'primary'"
                @keydown.enter.prevent="onSubmit"
                @keydown.esc.prevent="onReset"
              />
              <USelect
                v-model="selectedField"
                :items="fieldOptions"
                aria-label="Search field"
                size="sm"
                class="w-40 shrink-0"
              />
            </UFieldGroup>
          </UFormField>
          <p class="text-xs text-muted mt-1.5">
            {{ queryHelp }}
          </p>
        </div>
        <div class="sm:col-span-2 md:col-span-3 lg:col-span-2">
          <UFieldGroup
            size="sm"
            class="w-full md:mt-6"
          >
            <UButton
              label="Search"
              color="primary"
              size="sm"
              class="min-w-20 justify-center"
              :disabled="isLoading"
              @click="onSubmit"
            />
            <UTooltip
              text="Reset search filters"
              :delay-duration="0"
              :content="{ side: 'top' }"
            >
              <UButton
                icon="i-lucide-rotate-ccw"
                aria-label="Reset search filters"
                color="neutral"
                variant="outline"
                size="sm"
                class="shrink-0"
                :disabled="isLoading"
                @click="onReset"
              />
            </UTooltip>
          </UFieldGroup>
        </div>
      </div>
    </UCard>

    <UCard
      v-if="!hasError && hasRows"
      :ui="{ body: 'p-0 sm:p-0', footer: 'p-2 sm:px-2' }"
    >
      <UTable
        :data="rows"
        :columns="columns"
        :meta="{
          class: {
            tr: 'app-table-row cursor-pointer'
          }
        }"
        :on-select="onCollectionRowSelect"
        :ui="{
          th: 'px-4 py-3 text-sm text-highlighted',
          td: 'px-4 py-3 text-sm text-muted whitespace-nowrap'
        }"
      >
        <template #name-cell="{ row }">
          <UButton
            :label="row.original.name || 'N/A'"
            :to="`/collections/${row.original.id}`"
            color="primary"
            variant="link"
            size="xs"
            class="p-0 leading-none"
          />
        </template>

        <template #status-cell="{ row }">
          <UBadge
            :label="row.original.status.toUpperCase()"
            :color="statusColor(row.original.status)"
            variant="subtle"
            size="sm"
          />
        </template>

        <template #startedAt-cell="{ row }">
          {{ formatDateTime(row.original.startedAt) }}
        </template>

        <template #completedAt-cell="{ row }">
          {{ formatDateTime(row.original.completedAt) }}
        </template>
      </UTable>

      <template #footer>
        <div
          v-if="showPager"
          class="flex justify-end gap-1.5"
        >
          <UButton
            icon="i-lucide-house"
            aria-label="First page"
            color="neutral"
            variant="outline"
            size="xs"
            :disabled="!canGoPrev || isLoading"
            @click="onGoHome"
          />
          <UButton
            label="Previous"
            icon="i-lucide-chevrons-left"
            color="neutral"
            variant="outline"
            size="xs"
            :disabled="!canGoPrev || isLoading"
            @click="onGoPrev"
          />
          <UButton
            label="Next"
            trailing-icon="i-lucide-chevrons-right"
            color="neutral"
            variant="outline"
            size="xs"
            :disabled="!canGoNext || isLoading"
            @click="onGoNext"
          />
        </div>
      </template>
    </UCard>

    <UAlert
      v-else-if="!hasError"
      color="info"
      variant="subtle"
      title="No results"
      description="No collections matched the current search criteria."
    />
  </AppPageContainer>
</template>
