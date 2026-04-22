import type { EnduroStoredPipeline } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import { normalizeRouteStringParam } from './route-params'

export class PipelineLoadError extends Error {
  override name = 'PipelineLoadError'
}

type PipelinePageData = {
  pipeline: EnduroStoredPipeline
  pipelineId: string
  processingConfigurations: string[]
}

export const usePipelinePageData = defineBasicLoader<PipelinePageData>(
  async (to, { signal }) => {
    const pipelineId = normalizeRouteStringParam(to.params.id)
    if (!pipelineId) {
      throw new PipelineLoadError('The pipeline identifier is invalid.')
    }

    const { $enduroApi } = useNuxtApp()

    try {
      const [pipeline, processingConfigurations] = await Promise.all([
        $enduroApi.pipelines.show(pipelineId, { signal }),
        $enduroApi.pipelines.processing(pipelineId, { signal }).catch(() => [])
      ])

      return {
        pipeline,
        pipelineId,
        processingConfigurations
      }
    } catch {
      throw new PipelineLoadError('Could not load the pipeline from the API.')
    }
  },
  {
    errors: [PipelineLoadError]
  }
)
