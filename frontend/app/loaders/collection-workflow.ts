import type { EnduroCollectionWorkflowStatus } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import { parseCollectionId } from './route-params'

export class CollectionWorkflowLoadError extends Error {
  override name = 'CollectionWorkflowLoadError'
}

export const useCollectionWorkflowData = defineBasicLoader<{ workflow: EnduroCollectionWorkflowStatus }>(
  async (to, { signal }) => {
    const collectionId = parseCollectionId(to.params.id)
    if (collectionId <= 0) {
      throw new CollectionWorkflowLoadError('The collection identifier is invalid.')
    }

    const { $enduroApi } = useNuxtApp()

    try {
      const workflow = await $enduroApi.collections.workflow(collectionId, { signal })
      return { workflow }
    } catch {
      throw new CollectionWorkflowLoadError('The workflow history is not available.')
    }
  },
  {
    errors: [CollectionWorkflowLoadError]
  }
)
