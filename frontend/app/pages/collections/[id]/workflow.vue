<script setup lang="ts">
import type { ParsedWorkflowHistoryEvent } from '~/utils/workflow-history'
import {
  isWorkflowExecutionTerminalStatus,
  workflowActivityStatusColor,
  workflowExecutionStatusColor
} from '~/utils/workflow-summary'

const {
  collection,
  errorMessage,
  hasLoaded,
  isLoading,
  parsedWorkflow,
  loadWorkflow
} = useCollectionWorkflow()
const backToCollectionsRoute = useCollectionsListLocation()
const autoReload = useCollectionWorkflowAutoReload()
const selectedHistoryEvent = ref<ParsedWorkflowHistoryEvent | null>(null)
const historyDetailOpen = ref(false)

const AUTO_RELOAD_INTERVAL_MS = 5000
let autoReloadTimer: number | null = null
const shouldAutoReload = computed(() => (
  autoReload.value && !isWorkflowExecutionTerminalStatus(parsedWorkflow.value.status)
))

const metadataItems = computed(() => {
  const workflow = parsedWorkflow.value
  const items: Array<{ key: string, label: string, slot?: string, value?: string }> = [
    { key: 'workflowId', label: 'ID', slot: 'workflowId' },
    { key: 'runId', label: 'Run ID', slot: 'runId' },
    { key: 'status', label: 'Status', slot: 'status' }
  ]

  if (workflow.startedAt) {
    items.push({ key: 'startedAt', label: 'Started', value: formatDateTime(workflow.startedAt) })
  }

  if (workflow.completedAt) {
    items.push({ key: 'completedAt', label: 'Completed', slot: 'completedAt' })
  }

  return items
})

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

function formatDateTime(value: string | null): string {
  if (!value) return 'N/A'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return 'N/A'
  return dateTimeFormatter.format(parsed)
}

function formatDuration(startedAt: string | null, completedAt: string | null): string {
  if (!startedAt || !completedAt) return ''

  const start = new Date(startedAt)
  const end = new Date(completedAt)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) return ''

  const seconds = Math.max(0, Math.round((end.getTime() - start.getTime()) / 1000))
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const remainingSeconds = seconds % 60

  const segments: string[] = []
  if (hours) segments.push(`${hours}h`)
  if (minutes) segments.push(`${minutes}m`)
  if (!hours && !minutes) segments.push(`${remainingSeconds}s`)

  return segments.join(' ')
}

function isVerboseHistoryDescription(description: string): boolean {
  return description.length > 72 || description.includes('\n') || description.trimStart().startsWith('{')
}

function openHistoryDetail(event: ParsedWorkflowHistoryEvent) {
  selectedHistoryEvent.value = event
  historyDetailOpen.value = true
}

function clearAutoReloadTimer() {
  if (!autoReloadTimer) return

  window.clearInterval(autoReloadTimer)
  autoReloadTimer = null
}

function syncAutoReloadTimer() {
  clearAutoReloadTimer()

  if (!shouldAutoReload.value) return

  autoReloadTimer = window.setInterval(() => {
    if (isLoading.value) return
    void loadWorkflow(true)
  }, AUTO_RELOAD_INTERVAL_MS)
}

watch([
  autoReload,
  () => parsedWorkflow.value.status
], () => {
  syncAutoReloadTimer()
})

onMounted(() => {
  syncAutoReloadTimer()
})

onBeforeUnmount(() => {
  clearAutoReloadTimer()
})
</script>

<script lang="ts">
export { useCollectionWorkflowData } from '~/loaders/collection-workflow'
</script>

<template>
  <div
    v-if="errorMessage"
    class="space-y-4"
  >
    <UAlert
      color="warning"
      variant="subtle"
      title="Workflow unavailable"
      :description="errorMessage"
    >
      <template #actions>
        <div class="flex gap-2">
          <UButton
            label="Back to overview"
            :to="collection ? `/collections/${collection.id}` : backToCollectionsRoute"
            color="warning"
            variant="outline"
            size="sm"
          />
          <UButton
            label="Retry"
            color="warning"
            size="sm"
            :loading="isLoading"
            @click="loadWorkflow(true)"
          />
        </div>
      </template>
    </UAlert>
  </div>

  <div
    v-else-if="!hasLoaded && isLoading"
    class="space-y-4"
  >
    <UCard>
      <div class="space-y-3">
        <USkeleton class="h-5 w-40" />
        <USkeleton class="h-4 w-full" />
        <USkeleton class="h-4 w-5/6" />
        <USkeleton class="h-4 w-2/3" />
      </div>
    </UCard>
  </div>

  <div
    v-else
    class="space-y-4"
  >
    <UAlert
      color="neutral"
      variant="subtle"
      icon="i-lucide-workflow"
      title="Need deeper workflow details?"
      description="This view is intentionally lightweight. For full event payloads, cross-linked activity inspection, and deeper workflow debugging, use Temporal UI and search for this execution by the workflow ID or run ID shown below."
    />

    <UAlert
      v-if="parsedWorkflow.workflowError"
      color="error"
      variant="subtle"
      title="Workflow failure"
      :description="parsedWorkflow.workflowError"
    >
      <template #actions>
        <span class="text-xs text-muted">The workflow completed with an error.</span>
      </template>
    </UAlert>

    <UAlert
      v-else-if="parsedWorkflow.activityError"
      color="warning"
      variant="subtle"
      title="Activity error(s)"
      description="At least one activity has failed."
    />

    <UCard :ui="{ body: 'p-0 sm:p-0' }">
      <AppMetadataList
        :items="metadataItems"
        label-width="10rem"
      >
        <template #workflowId>
          <AppUuid :value="collection?.workflowId" />
        </template>

        <template #runId>
          <AppUuid :value="collection?.runId" />
        </template>

        <template #status>
          <UBadge
            :label="parsedWorkflow.status.toUpperCase()"
            :color="workflowExecutionStatusColor(parsedWorkflow.status)"
            variant="subtle"
          />
        </template>

        <template #completedAt>
          {{ formatDateTime(parsedWorkflow.completedAt) }}
          <span
            v-if="parsedWorkflow.startedAt"
            class="text-muted"
          >
            (took {{ formatDuration(parsedWorkflow.startedAt, parsedWorkflow.completedAt) }})
          </span>
        </template>
      </AppMetadataList>
    </UCard>

    <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
      <template #header>
        <h3 class="font-semibold">
          Activity summary
        </h3>
      </template>

      <AppWorkflowActivityList
        v-if="parsedWorkflow.activities.length"
        :activities="parsedWorkflow.activities"
        :format-date-time="formatDateTime"
        :status-color="workflowActivityStatusColor"
      />
      <div
        v-else
        class="px-4 py-4 text-sm text-muted sm:px-5"
      >
        No workflow activities were returned.
      </div>
    </UCard>

    <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
      <template #header>
        <h3 class="font-semibold">
          History
        </h3>
      </template>

      <div
        v-if="parsedWorkflow.events.length"
        class="overflow-x-auto"
      >
        <table class="min-w-full table-fixed">
          <colgroup>
            <col class="w-[5.5rem]">
            <col>
            <col class="w-[13rem]">
          </colgroup>
          <thead class="bg-elevated/30">
            <tr class="border-b border-default">
              <th
                scope="col"
                class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
              >
                ID
              </th>
              <th
                scope="col"
                class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
              >
                Event
              </th>
              <th
                scope="col"
                class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
              >
                Time
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-default">
            <tr
              v-for="event in parsedWorkflow.events"
              :key="event.id ?? `${event.type}-${event.eventTime}`"
              class="app-table-row"
            >
              <td class="px-4 py-3 align-top text-sm sm:px-5">
                <UBadge
                  v-if="event.id !== null"
                  :label="`#${event.id}`"
                  color="neutral"
                  variant="outline"
                  size="sm"
                />
                <span
                  v-else
                  class="text-muted"
                >N/A</span>
              </td>
              <td class="px-4 py-3 align-top sm:px-5">
                <div class="min-w-0 space-y-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="break-words font-medium text-highlighted">
                      {{ event.type }}
                    </span>
                    <UBadge
                      v-if="event.activityName"
                      :label="event.activityName"
                      color="primary"
                      variant="subtle"
                      size="sm"
                    />
                  </div>
                  <code
                    v-if="event.description && !isVerboseHistoryDescription(event.description)"
                    class="block whitespace-pre-wrap break-words rounded-md bg-elevated/60 px-2 py-1 text-xs text-toned"
                  >{{ event.description }}</code>
                  <div
                    v-else-if="event.description"
                    class="space-y-2"
                  >
                    <UButton
                      label="View details"
                      color="neutral"
                      variant="outline"
                      size="xs"
                      @click="openHistoryDetail(event)"
                    />
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 align-top text-sm text-muted whitespace-nowrap sm:px-5">
                {{ event.eventTime ? formatDateTime(event.eventTime) : 'Time unavailable' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div
        v-else
        class="px-4 py-4 text-sm text-muted sm:px-5"
      >
        No workflow history was returned.
      </div>
    </UCard>

    <UModal
      v-model:open="historyDetailOpen"
      fullscreen
      scrollable
      :title="selectedHistoryEvent?.type ?? 'History event details'"
      :description="selectedHistoryEvent?.eventTime ? formatDateTime(selectedHistoryEvent.eventTime) : 'Time unavailable'"
      :ui="{
        body: 'p-0 sm:p-0'
      }"
      @after:leave="selectedHistoryEvent = null"
    >
      <template #body>
        <div
          v-if="selectedHistoryEvent"
          class="space-y-6 p-4 sm:p-6 lg:p-8"
        >
          <UCard :ui="{ body: 'p-0 sm:p-0' }">
            <AppMetadataList
              :items="[
                { key: 'id', label: 'ID', value: selectedHistoryEvent.id !== null ? `#${selectedHistoryEvent.id}` : 'N/A' },
                { key: 'event', label: 'Event', value: selectedHistoryEvent.type },
                ...(selectedHistoryEvent.activityName ? [{ key: 'activity', label: 'Activity', slot: 'activity' }] : []),
                { key: 'time', label: 'Time', value: selectedHistoryEvent.eventTime ? formatDateTime(selectedHistoryEvent.eventTime) : 'Time unavailable' }
              ]"
              label-width="8rem"
            >
              <template #activity>
                <UBadge
                  :label="selectedHistoryEvent.activityName ?? ''"
                  color="primary"
                  variant="subtle"
                  size="sm"
                />
              </template>
            </AppMetadataList>
          </UCard>

          <UCard :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }">
            <template #header>
              <h3 class="font-semibold">
                Details
              </h3>
            </template>

            <pre class="overflow-x-auto whitespace-pre-wrap break-words px-4 py-4 font-mono text-sm text-toned sm:px-5">{{ selectedHistoryEvent.description }}</pre>
          </UCard>
        </div>
      </template>
    </UModal>
  </div>
</template>
