import type { BulkStatusResult } from '~/openapi-generator'
import { useNuxtApp } from '#app'
import { defineBasicLoader } from 'vue-router/experimental'

type CollectionBulkStatusData = {
  errorMessage: string
  status: BulkStatusResult
}

function createDefaultStatus(): BulkStatusResult {
  return {
    running: false
  }
}

export const useCollectionBulkStatusData = defineBasicLoader<CollectionBulkStatusData>(
  async (_to, { signal }) => {
    const { $enduroApi } = useNuxtApp()

    try {
      const status = await $enduroApi.collections.bulkStatus({ signal })
      return {
        errorMessage: '',
        status
      }
    } catch {
      return {
        errorMessage: 'Could not load bulk operation status.',
        status: createDefaultStatus()
      }
    }
  }
)
