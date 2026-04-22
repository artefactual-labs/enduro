import type { EnduroStoredPipeline } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'

export class PipelinesListLoadError extends Error {
  override name = 'PipelinesListLoadError'
}

export const usePipelinesListData = defineBasicLoader<{ pipelines: EnduroStoredPipeline[] }>(
  async (_to, { signal }) => {
    const { $enduroApi } = useNuxtApp()

    try {
      const pipelines = await $enduroApi.pipelines.list({ status: true }, { signal })
      return { pipelines }
    } catch {
      throw new PipelinesListLoadError('Could not load pipeline status.')
    }
  },
  {
    errors: [PipelinesListLoadError]
  }
)
