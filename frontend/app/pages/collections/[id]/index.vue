<script setup lang="ts">
import type { EnduroDetailedStoredCollection } from '~/openapi-generator'

const {
  activeAction,
  actionErrorMessage,
  canCancel,
  canDelete,
  canRetry,
  collection,
  errorMessage,
  hasLoaded,
  isLoading,
  isPending,
  pipeline,
  retryModeMessage,
  cancel,
  decide,
  dismissRetryModeMessage,
  reload,
  remove,
  retry
} = useCollectionDetails()
const backToCollectionsRoute = useCollectionsListLocation()
const deleteDialogOpen = ref(false)

const isBusy = computed(() => isLoading.value || activeAction.value !== null)

const overviewItems = computed(() => {
  if (!collection.value) return []

  const items: Array<{ key: string, label: string, slot?: string, value?: string, valueClass?: string }> = [
    { key: 'status', label: 'Status', slot: 'status' },
    { key: 'created', label: 'Created', value: formatDateTime(collection.value.createdAt) },
    { key: 'started', label: 'Started', value: collection.value.startedAt ? formatDateTime(collection.value.startedAt) : 'Not started yet.' }
  ]

  if (collection.value.originalId) {
    items.splice(1, 0, { key: 'originalId', label: 'Original ID', value: collection.value.originalId })
  }

  if (collection.value.status === 'done' && collection.value.completedAt) {
    items.push({ key: 'stored', label: 'Stored', slot: 'stored' })
  }

  if (collection.value.transferId) {
    items.push({ key: 'transferId', label: 'Transfer', slot: 'transferId' })
  }

  if (collection.value.aipId) {
    items.push({ key: 'aipId', label: 'AIP', slot: 'aipId' })
  }

  return items
})

const workflowItems = computed(() => [
  { key: 'workflowId', label: 'ID', slot: 'workflowId' },
  { key: 'runId', label: 'Run ID', slot: 'runId' }
])

const pipelineItems = computed(() => {
  if (!collection.value?.pipelineId) return []

  const items: Array<{ key: string, label: string, slot?: string, value?: string }> = [
    { key: 'pipelineId', label: 'ID', slot: 'pipelineId' }
  ]

  if (pipeline.value) {
    items.unshift({ key: 'name', label: 'Name', value: pipeline.value.name })
    items.push({ key: 'capacity', label: 'Capacity', slot: 'capacity' })
  }

  return items
})

function statusColor(status: EnduroDetailedStoredCollection['status']) {
  if (status === 'done') return 'success'
  if (status === 'error') return 'error'
  if (status === 'in progress') return 'warning'
  if (status === 'pending' || status === 'queued') return 'info'
  return 'neutral'
}

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

onUnmounted(() => {
  dismissRetryModeMessage()
})

function formatDateTime(value: Date | string | null | undefined): string {
  if (!value) return 'N/A'
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return 'N/A'
  return dateTimeFormatter.format(date)
}

function formatDuration(from: Date | string | null | undefined, to: Date | string | null | undefined): string {
  if (!from || !to) return ''

  const start = from instanceof Date ? from : new Date(from)
  const end = to instanceof Date ? to : new Date(to)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return ''

  let remainingSeconds = Math.max(0, Math.round((end.getTime() - start.getTime()) / 1000))
  const hours = Math.floor(remainingSeconds / 3600)
  remainingSeconds -= hours * 3600
  const minutes = Math.floor(remainingSeconds / 60)
  const seconds = remainingSeconds - minutes * 60

  const segments: string[] = []
  if (hours) segments.push(`${hours}h`)
  if (minutes) segments.push(`${minutes}m`)
  if (!hours && !minutes) segments.push(`${seconds}s`)

  return segments.join(' ')
}

async function confirmDelete() {
  const completed = await remove()
  if (!completed) {
    deleteDialogOpen.value = false
  }
}
</script>

<template>
  <div>
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
              :loading="activeAction === 'reload'"
              @click="reload"
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
      class="grid grid-cols-1 xl:grid-cols-12 gap-6"
    >
      <div class="xl:col-span-7 space-y-4">
        <UAlert
          v-if="actionErrorMessage"
          color="error"
          variant="subtle"
          title="Action failed"
          :description="actionErrorMessage"
        />

        <UAlert
          v-if="retryModeMessage"
          close
          color="info"
          variant="subtle"
          :description="retryModeMessage"
          @update:open="open => { if (!open) dismissRetryModeMessage() }"
        />

        <UAlert
          v-if="isPending"
          color="warning"
          variant="subtle"
          title="Awaiting decision"
        >
          <template #description>
            <p class="mb-3">
              An activity has failed irremediably. More information can be found under the Workflow tab.
            </p>
            <div class="flex gap-2">
              <UButton
                label="Abandon"
                color="neutral"
                variant="outline"
                size="sm"
                :loading="activeAction === 'decide-abandon'"
                :disabled="isBusy"
                @click="decide('ABANDON')"
              />
              <UButton
                label="Retry"
                color="info"
                size="sm"
                :loading="activeAction === 'decide-retry'"
                :disabled="isBusy"
                @click="decide('RETRY_ONCE')"
              />
            </div>
          </template>
        </UAlert>

        <UCard :ui="{ body: 'p-0 sm:p-0' }">
          <AppMetadataList :items="overviewItems">
            <template #status>
              <UBadge
                :label="collection.status.toUpperCase()"
                :color="statusColor(collection.status)"
                variant="subtle"
              />
            </template>

            <template #stored>
              {{ formatDateTime(collection.completedAt) }}
              <span
                v-if="collection.startedAt"
                class="text-muted"
              >
                (took {{ formatDuration(collection.startedAt, collection.completedAt) }})
              </span>
            </template>

            <template #transferId>
              <AppUuid :value="collection.transferId" />
            </template>

            <template #aipId>
              <AppUuid :value="collection.aipId" />
            </template>
          </AppMetadataList>
        </UCard>

        <div class="flex items-center justify-end gap-3 text-sm">
          <UButton
            v-if="canDelete"
            label="Delete"
            color="error"
            variant="outline"
            size="sm"
            :loading="activeAction === 'delete'"
            :disabled="isBusy"
            @click="deleteDialogOpen = true"
          />
        </div>
      </div>

      <div class="xl:col-span-5 space-y-4">
        <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
          <template #header>
            <h3 class="font-semibold">
              Pipeline
            </h3>
          </template>

          <template v-if="collection.pipelineId">
            <AppMetadataList
              :items="pipelineItems"
              layout="stacked"
            >
              <template #pipelineId>
                <AppUuid :value="collection.pipelineId" />
              </template>

              <template #capacity>
                <AppPipelineCapacity
                  :current="pipeline?.current"
                  :capacity="pipeline?.capacity"
                />
              </template>
            </AppMetadataList>

            <template v-if="pipeline">
              <div class="border-t border-default px-4 py-4 sm:px-5">
                <UButton
                  label="Status"
                  :to="`/pipelines/${collection.pipelineId}`"
                  size="sm"
                  color="neutral"
                  variant="outline"
                />
              </div>
            </template>
            <div
              v-else
              class="border-t border-default px-4 py-4 text-sm text-muted sm:px-5"
            >
              Pipeline details are not available.
            </div>
          </template>
          <p
            v-else
            class="px-4 py-4 text-sm text-muted sm:px-5"
          >
            Not identified yet.
          </p>
        </UCard>

        <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
          <template #header>
            <h3 class="font-semibold">
              Workflow
            </h3>
          </template>

          <AppMetadataList
            :items="workflowItems"
            layout="stacked"
          >
            <template #workflowId>
              <AppUuid :value="collection.workflowId" />
            </template>

            <template #runId>
              <AppUuid :value="collection.runId" />
            </template>
          </AppMetadataList>

          <div class="border-t border-default px-4 py-4 sm:px-5">
            <div class="flex gap-2">
              <UButton
                label="Status"
                :to="`/collections/${collection.id}/workflow`"
                size="sm"
                color="neutral"
                variant="outline"
              />
              <UButton
                v-if="canRetry"
                label="Retry"
                size="sm"
                color="info"
                :loading="activeAction === 'retry'"
                :disabled="isBusy"
                @click="retry"
              />
              <UButton
                v-if="canCancel"
                label="Cancel"
                size="sm"
                color="neutral"
                :loading="activeAction === 'cancel'"
                :disabled="isBusy"
                @click="cancel"
              />
            </div>
          </div>
        </UCard>
      </div>
    </div>

    <AppConfirmDialog
      v-model:open="deleteDialogOpen"
      title="Delete collection?"
      description="This operation cannot be reversed."
      confirm-label="Delete"
      confirm-color="error"
      :pending="activeAction === 'delete'"
      @confirm="confirmDelete"
    />
  </div>
</template>
