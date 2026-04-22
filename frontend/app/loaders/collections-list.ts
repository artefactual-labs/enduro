import type { EnduroStoredCollection } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'
import { buildCollectionListRequest } from './collections-list.helpers'

type CollectionsListData = {
  items: EnduroStoredCollection[]
  nextCursor: string | null
  invalidQuery: boolean
}

export class CollectionsListLoadError extends Error {
  override name = 'CollectionsListLoadError'
}

export const useCollectionsListData = defineBasicLoader<CollectionsListData>(
  async (to, { signal }) => {
    const { invalidQuery, request } = buildCollectionListRequest(to.query as Record<string, unknown>)
    if (invalidQuery) {
      return {
        items: [],
        nextCursor: null,
        invalidQuery: true
      }
    }

    const { $enduroApi } = useNuxtApp()

    try {
      const response = await $enduroApi.collections.list(request, { signal })
      return {
        items: response.items,
        nextCursor: response.nextCursor ?? null,
        invalidQuery: false
      }
    } catch {
      throw new CollectionsListLoadError('Could not load collections from the API.')
    }
  },
  {
    errors: [CollectionsListLoadError]
  }
)
