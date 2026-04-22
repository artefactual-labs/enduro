import type { BatchHintsResult, BatchStatusResult, EnduroStoredPipeline } from '../openapi-generator'

type SelectOption = {
  label: string
  value: string
}

export type BatchPageData = {
  hints: BatchHintsResult
  hintsErrorMessage: string
  pipelines: EnduroStoredPipeline[]
  pipelinesErrorMessage: string
  processingOptions: SelectOption[]
  processingErrorMessage: string
  selectedPipelineId: string
}

export function createDefaultBatchHints(): BatchHintsResult {
  return {
    completedDirs: []
  }
}

export function createDefaultBatchStatus(): BatchStatusResult {
  return {
    running: false
  }
}

export function toSelectOptions(values: string[]): SelectOption[] {
  return values.map(value => ({
    label: value,
    value
  }))
}

export function resolveBatchPageData(input: {
  loadedHints: PromiseSettledResult<BatchHintsResult>
  loadedPipelines: PromiseSettledResult<EnduroStoredPipeline[]>
  loadedProcessingOptions: PromiseSettledResult<string[]>
  selectedPipelineId: string
}): BatchPageData {
  return {
    hints: input.loadedHints.status === 'fulfilled' ? input.loadedHints.value : createDefaultBatchHints(),
    hintsErrorMessage: input.loadedHints.status === 'fulfilled' ? '' : 'Could not load batch path hints.',
    pipelines: input.loadedPipelines.status === 'fulfilled' ? input.loadedPipelines.value : [],
    pipelinesErrorMessage: input.loadedPipelines.status === 'fulfilled' ? '' : 'Could not load configured pipelines.',
    processingOptions: input.loadedProcessingOptions.status === 'fulfilled'
      ? toSelectOptions(input.loadedProcessingOptions.value)
      : [],
    processingErrorMessage: input.loadedProcessingOptions.status === 'fulfilled'
      ? ''
      : 'Could not load processing configurations for the selected pipeline.',
    selectedPipelineId: input.selectedPipelineId
  }
}
