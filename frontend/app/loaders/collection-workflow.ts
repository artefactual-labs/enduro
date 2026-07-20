import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import {
  type CollectionWorkflowData,
  loadCollectionWorkflowSources
} from './collection-workflow.helpers'
import { parseCollectionId } from './route-params'

export class CollectionWorkflowLoadError extends Error {
  override name = 'CollectionWorkflowLoadError'
}

export const useCollectionWorkflowData = defineBasicLoader<CollectionWorkflowData>(
  async (to, { signal }) => {
    const collectionId = parseCollectionId(to.params.id)
    if (collectionId <= 0) {
      throw new CollectionWorkflowLoadError('The collection identifier is invalid.')
    }

    const { $enduroApi } = useNuxtApp()

    return loadCollectionWorkflowSources($enduroApi.collections, collectionId, signal)
  },
  {
    errors: [CollectionWorkflowLoadError]
  }
)
