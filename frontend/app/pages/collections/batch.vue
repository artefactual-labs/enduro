<script setup lang="ts">
const {
  canSubmit,
  completedDir,
  destinationMode,
  destinationModeOptions,
  depth,
  excludeHiddenFiles,
  hasKnownCompletedDirs,
  hints,
  hintsErrorMessage,
  isLoadingHints,
  isLoadingPipelines,
  isLoadingProcessing,
  isLoadingStatus,
  isRunning,
  isSubmitting,
  path,
  pipelineOptions,
  pipelinesErrorMessage,
  processNameMetadata,
  processingErrorMessage,
  processingOptions,
  rejectDuplicates,
  retentionPeriod,
  selectedPipelineId,
  selectedProcessingConfig,
  selectedTransferType,
  status,
  statusErrorMessage,
  loadStatus,
  submit,
  submitErrorMessage,
  submitSuccessMessage,
  transferOptions,
  useCompletedDirHint
} = useBatchImport()

useSeoMeta({
  title: 'Batch import'
})
</script>

<script lang="ts">
export { useBatchPageData, useBatchStatusData } from '~/loaders/batch-page'
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
      title="Batch submit failed"
      :description="submitErrorMessage"
    />

    <UAlert
      v-if="submitSuccessMessage"
      color="success"
      variant="subtle"
      title="Batch submitted"
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
              Batch running
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
          Batch operation is still running. Status refreshes automatically every second.
        </p>

        <dl class="grid grid-cols-3 gap-y-2 text-sm">
          <dt class="text-muted">
            Status
          </dt>
          <dd class="col-span-2">
            {{ status.status || 'running' }}
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
            name="i-lucide-folder-input"
            class="size-7 shrink-0 text-primary"
          />
          <div class="space-y-1">
            <p class="text-xs font-medium uppercase tracking-[0.2em] text-muted">
              Batch import
            </p>
            <h1 class="text-xl font-semibold text-highlighted">
              Start a new batch
            </h1>
          </div>
        </div>
      </template>

      <div class="space-y-5">
        <p class="text-sm text-muted">
          Submit a new batch by choosing the source path, optional pipeline and processing configuration, and the destination behavior for processed transfers.
        </p>

        <div class="grid grid-cols-1 gap-4">
          <UFormField
            label="Path"
            description="Select the path for the batch"
            required
          >
            <UInput
              v-model="path"
              placeholder="/path/to/transfers"
              class="w-full"
            />
          </UFormField>

          <UFormField
            label="Pipeline"
            description="Optional. Choose one of the configured pipelines"
          >
            <USelect
              v-model="selectedPipelineId"
              :items="pipelineOptions"
              :loading="isLoadingPipelines"
              placeholder="Select a pipeline"
              class="w-full"
            />
          </UFormField>

          <UAlert
            v-if="pipelinesErrorMessage"
            color="warning"
            variant="subtle"
            :description="pipelinesErrorMessage"
          />

          <UFormField
            v-if="selectedPipelineId"
            label="Processing configuration"
            description="Optional. Choose one of the processing configurations available"
          >
            <USelect
              v-model="selectedProcessingConfig"
              :items="processingOptions"
              :loading="isLoadingProcessing"
              placeholder="Select a processing configuration"
              class="w-full"
            />
          </UFormField>

          <UAlert
            v-if="processingErrorMessage"
            color="warning"
            variant="subtle"
            :description="processingErrorMessage"
          />

          <UFormField
            label="Transfer type"
            description="Optional. Choose the transfer type, with the default being standard"
          >
            <USelect
              v-model="selectedTransferType"
              :items="transferOptions"
              placeholder="Standard"
              class="w-full"
            />
          </UFormField>

          <div class="flex flex-col gap-3 md:flex-row md:flex-wrap md:items-center md:gap-x-6">
            <UCheckbox
              v-model="rejectDuplicates"
              label="Reject transfers with duplicate names"
            />
            <UCheckbox
              v-model="excludeHiddenFiles"
              label="Exclude hidden files"
            />
            <UCheckbox
              v-model="processNameMetadata"
              label="Process transfer name metadata"
            />
          </div>

          <UFormField
            label="Depth"
            description="Depth where SIPs reside in the hierarchy"
          >
            <UInput
              v-model.number="depth"
              type="number"
              min="0"
              step="1"
              class="w-full"
            />
          </UFormField>

          <UFormField
            label="Destination behavior"
            description="Choose how processed transfers are handled after completion"
          >
            <USelect
              v-model="destinationMode"
              :items="destinationModeOptions"
              class="w-full"
            />
          </UFormField>

          <UFormField
            v-if="destinationMode === 'completed-dir'"
            label="Completed directory"
            description="Optional. The path where transfers are moved when processing completes successfully"
          >
            <UInput
              v-model="completedDir"
              placeholder="/path/to/completed"
              class="w-full"
            />
          </UFormField>

          <div
            v-if="destinationMode === 'completed-dir' && hasKnownCompletedDirs"
            class="space-y-2"
          >
            <p class="text-xs font-medium uppercase tracking-[0.18em] text-muted">
              Known directories
            </p>
            <div class="flex flex-wrap gap-2">
              <UButton
                v-for="item in hints.completedDirs"
                :key="item"
                :label="item"
                color="neutral"
                variant="outline"
                size="xs"
                @click="useCompletedDirHint(item)"
              />
            </div>
          </div>

          <UFormField
            v-if="destinationMode === 'retention-period'"
            label="Retention period"
            description="Optional. Use Go-style durations such as 30m, 24h, or 2h30m"
          >
            <UInput
              v-model="retentionPeriod"
              placeholder="24h"
              class="w-full"
            />
          </UFormField>

          <UAlert
            v-if="hintsErrorMessage"
            color="warning"
            variant="subtle"
            :description="hintsErrorMessage"
          />
        </div>

        <div class="flex justify-end">
          <UButton
            label="Submit"
            color="primary"
            :loading="isSubmitting || isLoadingHints"
            :disabled="!canSubmit"
            @click="submit"
          />
        </div>
      </div>
    </UCard>
  </AppPageContainer>
</template>
