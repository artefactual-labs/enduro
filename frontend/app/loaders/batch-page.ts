import type { BatchStatusResult } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import { normalizeRouteStringParam } from './route-params'
import {
  createDefaultBatchStatus,
  type BatchPageData,
  resolveBatchPageData
} from './batch-page.helpers'

export type BatchStatusData = {
  errorMessage: string
  status: BatchStatusResult
}

export const useBatchPageData = defineBasicLoader<BatchPageData>(
  async (to, { signal }) => {
    const selectedPipelineId = normalizeRouteStringParam(to.query.pipeline)
    const { $enduroApi } = useNuxtApp()

    const [loadedPipelines, loadedHints, loadedProcessingOptions] = await Promise.allSettled([
      $enduroApi.pipelines.list({}, { signal }),
      $enduroApi.batches.hints({ signal }),
      selectedPipelineId
        ? $enduroApi.pipelines.processing(selectedPipelineId, { signal })
        : Promise.resolve([])
    ])

    return resolveBatchPageData({
      loadedHints,
      loadedPipelines,
      loadedProcessingOptions,
      selectedPipelineId
    })
  }
)

export const useBatchStatusData = defineBasicLoader<BatchStatusData>(
  async (_to, { signal }) => {
    const { $enduroApi } = useNuxtApp()

    try {
      const status = await $enduroApi.batches.status({ signal })
      return {
        errorMessage: '',
        status
      }
    } catch {
      return {
        errorMessage: 'Could not load batch status.',
        status: createDefaultBatchStatus()
      }
    }
  }
)
