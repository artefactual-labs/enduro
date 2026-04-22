import type { EnduroDetailedStoredCollection, EnduroStoredPipeline } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import { parseCollectionId } from './route-params'

export class CollectionDetailsLoadError extends Error {
  override name = 'CollectionDetailsLoadError'
}

type CollectionPageData = {
  collection: EnduroDetailedStoredCollection
  pipeline: EnduroStoredPipeline | null
}

export const useCollectionPageData = defineBasicLoader<CollectionPageData>(
  async (to, { signal }) => {
    const collectionId = parseCollectionId(to.params.id)
    if (collectionId <= 0) {
      throw new CollectionDetailsLoadError('The collection identifier is invalid.')
    }

    const { $enduroApi } = useNuxtApp()

    try {
      const collection = await $enduroApi.collections.show(collectionId, { signal })
      const pipeline = collection.pipelineId
        ? await $enduroApi.pipelines.show(collection.pipelineId, { signal }).catch(() => null)
        : null

      return {
        collection,
        pipeline
      }
    } catch {
      throw new CollectionDetailsLoadError('Could not load the collection from the API.')
    }
  },
  {
    errors: [CollectionDetailsLoadError]
  }
)
