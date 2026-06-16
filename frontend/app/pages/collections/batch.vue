<script setup lang="ts">
const {
  canBrowseBatchDirectories,
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
  showBatchStatus,
  status,
  statusErrorMessage,
  loadStatus,
  submit,
  submitErrorMessage,
  transferOptions,
  useBrowsedPath,
  useCompletedDirHint
} = useBatchImport()

const batchStatusLabel = computed(() => (
  isRunning.value ? 'RUNNING' : (status.value.status?.toUpperCase() || 'SUBMITTED')
))

const batchStatusColor = computed(() => {
  if (isRunning.value) return 'warning'

  switch (status.value.status?.toLowerCase()) {
    case 'completed':
      return 'success'
    case 'failed':
    case 'canceled':
    case 'terminated':
    case 'timed_out':
      return 'error'
    default:
      return 'neutral'
  }
})

const submitConfirmationOpen = ref(false)
const directoryBrowserOpen = ref(false)

const selectedProcessingConfigLabel = computed(() => {
  const option = processingOptions.value.find(item => item.value === selectedProcessingConfig.value)
  if (option?.label) return option.label
  if (selectedProcessingConfig.value) return selectedProcessingConfig.value

  return 'Default (none selected)'
})

const submitConfirmationPath = computed(() => path.value.trim())

function requestSubmitConfirmation() {
  if (!canSubmit.value) return
  submitConfirmationOpen.value = true
}

async function confirmSubmit() {
  await submit()
  submitConfirmationOpen.value = false
}

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
      v-if="statusErrorMessage"
      color="warning"
      variant="subtle"
      title="Status unavailable"
      :description="statusErrorMessage"
    />

    <UCard v-if="showBatchStatus">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="text-xs font-medium uppercase tracking-[0.2em] text-muted">
              {{ isRunning ? 'Active job' : 'Submission received' }}
            </p>
            <h1 class="text-xl font-semibold text-highlighted">
              {{ isRunning ? 'Batch running' : 'Batch submitted' }}
            </h1>
          </div>

          <UBadge
            :label="batchStatusLabel"
            :color="batchStatusColor"
            variant="solid"
          />
        </div>
      </template>

      <div class="space-y-4">
        <p class="text-sm text-muted">
          {{ isRunning ? 'Batch operation is still running. Status refreshes automatically every second.' : 'Enduro accepted the batch. Review the collections list to follow each transfer.' }}
        </p>

        <dl class="grid grid-cols-3 gap-y-2 text-sm">
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

        <div class="flex justify-end gap-2">
          <UButton
            label="Refresh now"
            color="neutral"
            variant="outline"
            size="sm"
            :loading="isLoadingStatus"
            @click="loadStatus"
          />
          <UButton
            to="/collections"
            label="View collections"
            color="primary"
            size="sm"
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
          Submit a new batch by choosing the source directory, optional pipeline and processing configuration, and the destination behavior for processed transfers.
        </p>

        <div class="grid grid-cols-1 gap-4">
          <UFormField
            label="Path"
            description="Select the parent directory that contains the batch transfers"
            required
          >
            <div class="flex gap-2">
              <UInput
                v-model="path"
                placeholder="/path/to/transfers"
                class="min-w-0 flex-1"
              />
              <UButton
                v-if="canBrowseBatchDirectories"
                icon="i-lucide-folder-search"
                label="Browse"
                color="neutral"
                variant="outline"
                @click="directoryBrowserOpen = true"
              />
            </div>
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
            description="0 selects direct children of the batch path; 1 selects one level below. Enduro submits folders and non-hidden files at that level."
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
            @click="requestSubmitConfirmation"
          />
        </div>
      </div>
    </UCard>

    <AppConfirmDialog
      v-model:open="submitConfirmationOpen"
      title="Submit batch import?"
      description="Review the batch details before submitting."
      confirm-label="Submit"
      cancel-label="Cancel"
      modal-class="max-w-lg"
      :pending="isSubmitting"
      @confirm="confirmSubmit"
    >
      <p class="text-sm text-muted">
        You're submitting this directory as a batch transfer.
      </p>

      <div class="space-y-4">
        <div class="space-y-2">
          <p class="text-xs font-medium uppercase tracking-[0.14em] text-muted">
            Path
          </p>
          <pre class="overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-elevated px-3 py-2 font-mono text-xs leading-relaxed text-highlighted">{{ submitConfirmationPath }}</pre>
        </div>

        <div class="space-y-2">
          <p class="text-xs font-medium uppercase tracking-[0.14em] text-muted">
            Processing configuration
          </p>
          <p class="rounded-md bg-elevated px-3 py-2 text-sm text-highlighted">
            {{ selectedProcessingConfigLabel }}
          </p>
        </div>
      </div>
    </AppConfirmDialog>

    <BatchDirectoryBrowser
      v-if="canBrowseBatchDirectories"
      v-model:open="directoryBrowserOpen"
      @select="useBrowsedPath"
    />
  </AppPageContainer>
</template>
