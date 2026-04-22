<script setup lang="ts">
import type { EnduroDetailedStoredCollection } from '~/openapi-generator'

const {
  collection,
  errorMessage,
  hasLoaded,
  isLoading,
  loadCollection
} = useCollectionDetails()

const backToCollectionsRoute = useCollectionsListLocation()

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

function formatDateTime(value: Date | string | null | undefined): string {
  if (!value) return 'N/A'
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return 'N/A'
  return dateTimeFormatter.format(date)
}

function statusVariant(status: EnduroDetailedStoredCollection['reconciliationStatus']): 'success' | 'warning' | 'error' | 'neutral' {
  switch (status) {
    case 'complete':
      return 'success'
    case 'partial':
      return 'warning'
    case 'unknown':
      return 'error'
    default:
      return 'neutral'
  }
}

const summaryStatus = computed(() => {
  if (!collection.value || !collection.value.aipId) return 'not available'
  return collection.value.reconciliationStatus || 'not available'
})

const summaryColor = computed(() => {
  if (summaryStatus.value === 'not available') return 'neutral'
  return statusVariant(summaryStatus.value)
})

const summaryLabel = computed(() => summaryStatus.value.toUpperCase())

const metadataItems = computed(() => {
  if (!collection.value) return []

  const items: Array<{ key: string, label: string, slot?: string, value?: string, valueClass?: string }> = []

  if (collection.value.aipId) {
    items.push({ key: 'aipId', label: 'AIP', slot: 'aipId' })
  }

  if (collection.value.aipStoredAt) {
    items.push({ key: 'aipStoredAt', label: 'Primary AIP stored at', value: formatDateTime(collection.value.aipStoredAt) })
  }

  if (collection.value.reconciliationCheckedAt) {
    items.push({ key: 'checkedAt', label: 'Last checked at', value: formatDateTime(collection.value.reconciliationCheckedAt) })
  }

  if (collection.value.reconciliationError) {
    items.push({ key: 'reconciliationError', label: 'Reconciliation error', value: collection.value.reconciliationError, valueClass: 'text-error' })
  }

  return items
})

const summaryText = computed(() => {
  if (!collection.value || !collection.value.aipId) {
    return 'Storage reconciliation is not available yet. An AIP has not been created.'
  }

  switch (collection.value.reconciliationStatus) {
    case 'complete':
      return 'Storage is complete.'
    case 'partial':
      return 'Primary AIP exists, but required storage is incomplete.'
    case 'unknown':
      return 'Storage state could not be determined.'
    case 'pending':
      return 'Storage reconciliation has not produced a final result yet.'
    default:
      return 'No storage reconciliation has been recorded for this collection yet.'
  }
})
</script>

<template>
  <div
    v-if="errorMessage"
    class="space-y-4"
  >
    <UAlert
      color="warning"
      variant="subtle"
      title="Collection unavailable"
      :description="errorMessage"
    >
      <template #actions>
        <div class="flex gap-2">
          <UButton
            label="Back to collections"
            :to="backToCollectionsRoute"
            color="warning"
            variant="outline"
            size="sm"
          />
          <UButton
            label="Retry"
            color="warning"
            size="sm"
            :loading="isLoading"
            @click="loadCollection(true)"
          />
        </div>
      </template>
    </UAlert>
  </div>

  <div
    v-else-if="!collection && !hasLoaded"
    class="space-y-4"
  >
    <UCard>
      <div class="space-y-3">
        <USkeleton class="h-5 w-32" />
        <USkeleton class="h-4 w-full" />
        <USkeleton class="h-4 w-5/6" />
        <USkeleton class="h-4 w-2/3" />
      </div>
    </UCard>
  </div>

  <div
    v-else-if="collection"
    class="space-y-4"
  >
    <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
      <template #header>
        <div class="flex items-start justify-between gap-4">
          <div class="space-y-1">
            <p class="text-xs font-medium uppercase tracking-[0.12em] text-muted">
              Storage status
            </p>
            <h3 class="font-semibold text-highlighted">
              {{ summaryText }}
            </h3>
          </div>

          <UBadge
            :label="summaryLabel"
            :color="summaryColor"
            variant="subtle"
            size="sm"
          />
        </div>
      </template>

      <AppMetadataList :items="metadataItems">
        <template #aipId>
          <AppUuid :value="collection.aipId" />
        </template>
      </AppMetadataList>
    </UCard>
  </div>
</template>
