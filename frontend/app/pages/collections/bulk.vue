<script setup lang="ts">
const {
  canSubmit,
  hasCompletedRun,
  isLoadingStatus,
  isRunning,
  isSubmitting,
  lastRunFailed,
  loadStatus,
  operationOptions,
  selectedOperation,
  selectedStatus,
  size,
  status,
  statusErrorMessage,
  statusOptions,
  submit,
  submitErrorMessage,
  submitSuccessMessage
} = useCollectionBulk()

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

useSeoMeta({
  title: 'Bulk operation'
})
</script>

<script lang="ts">
export { useCollectionBulkStatusData } from '~/loaders/collection-bulk-status'
</script>

<template>
  <AppPageContainer>
    <UButton
      to="/collections"
      label="Back to collections"
      icon="i-lucide-arrow-left"
      color="neutral"
      variant="ghost"
    />

    <UAlert
      v-if="submitErrorMessage"
      color="error"
      variant="subtle"
      title="Bulk submit failed"
      :description="submitErrorMessage"
    />

    <UAlert
      v-if="submitSuccessMessage"
      color="success"
      variant="subtle"
      title="Bulk operation submitted"
      :description="submitSuccessMessage"
    />

    <UAlert
      v-if="statusErrorMessage"
      color="warning"
      variant="subtle"
      title="Status unavailable"
      :description="statusErrorMessage"
    />

    <UCard v-if="isRunning">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-xs font-medium uppercase tracking-[0.2em] text-muted">
              Active job
            </p>
            <h1 class="text-xl font-semibold text-highlighted">
              Bulk operation running
            </h1>
          </div>

          <UBadge
            label="RUNNING"
            color="warning"
            variant="subtle"
          />
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-sm text-muted">
          Bulk operation is still running. Status refreshes automatically every second.
        </p>

        <dl class="grid grid-cols-3 gap-y-2 text-sm">
          <dt class="text-muted">
            Status
          </dt>
          <dd class="col-span-2">
            {{ status.status || 'running' }}
          </dd>
          <dt class="text-muted">
            Started
          </dt>
          <dd class="col-span-2">
            {{ formatDateTime(status.startedAt) }}
          </dd>
          <dt class="text-muted">
            Workflow ID
          </dt>
          <dd class="col-span-2 break-all">
            <AppUuid :value="status.workflowId" />
          </dd>
          <dt class="text-muted">
            Run ID
          </dt>
          <dd class="col-span-2 break-all">
            <AppUuid :value="status.runId" />
          </dd>
        </dl>

        <div class="flex justify-end">
          <UButton
            label="Refresh now"
            color="neutral"
            variant="outline"
            size="sm"
            :loading="isLoadingStatus"
            @click="loadStatus"
          />
        </div>
      </div>
    </UCard>

    <UCard v-else>
      <template #header>
        <div class="flex items-center gap-3">
          <UIcon
            name="i-lucide-list-checks"
            class="size-7 shrink-0 text-primary"
          />
          <div class="space-y-1">
            <p class="text-xs font-medium uppercase tracking-[0.2em] text-muted">
              Bulk operation
            </p>
            <h1 class="text-xl font-semibold text-highlighted">
              Start a new bulk operation
            </h1>
          </div>
        </div>
      </template>

      <div class="space-y-5">
        <UAlert
          v-if="hasCompletedRun"
          :color="lastRunFailed ? 'warning' : 'success'"
          variant="subtle"
          :title="lastRunFailed ? 'Previous bulk ended with issues' : 'Previous bulk completed'"
          :description="`Last run completed with status '${status.status || 'unknown'}' at ${formatDateTime(status.closedAt)}.`"
        >
          <template
            v-if="lastRunFailed && status.workflowId"
            #actions
          >
            <AppUuid :value="status.workflowId" />
          </template>
        </UAlert>

        <p class="text-sm text-muted">
          Start a new bulk operation. The current parity scope matches the legacy UI and supports retrying collections in the error state.
        </p>

        <div class="grid grid-cols-1 gap-4">
          <UFormField
            label="Collection status filter"
            description="Select the status of the collections that you want to modify."
          >
            <USelect
              v-model="selectedStatus"
              :items="statusOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField
            label="Operation"
            description="Type of operation to be performed."
          >
            <USelect
              v-model="selectedOperation"
              :items="operationOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField
            label="Size"
            description="Optional. Maximum number of collections affected."
          >
            <UInput
              v-model.number="size"
              type="number"
              min="1"
              step="1"
              placeholder="No limit"
              class="w-full"
            />
          </UFormField>
        </div>

        <div class="flex justify-end">
          <UButton
            label="Submit"
            color="primary"
            :loading="isSubmitting"
            :disabled="!canSubmit"
            @click="submit"
          />
        </div>
      </div>
    </UCard>
  </AppPageContainer>
</template>
