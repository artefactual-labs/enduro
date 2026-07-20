import type {
  EnduroCollectionStatusHistory,
  EnduroCollectionWorkflowStatus
} from '~/openapi-generator'
import type { EnduroApiClient } from '~/utils/enduro-api-client'

export type CollectionWorkflowData = {
  workflow: EnduroCollectionWorkflowStatus | null
  workflowError: string
  statusHistory: EnduroCollectionStatusHistory | null
  statusHistoryError: string
}

type CollectionWorkflowClient = Pick<EnduroApiClient['collections'], 'statusHistory' | 'workflow'>

export async function loadCollectionWorkflowSources(
  collections: CollectionWorkflowClient,
  collectionId: number,
  signal?: AbortSignal
): Promise<CollectionWorkflowData> {
  const [workflowResult, statusHistoryResult] = await Promise.allSettled([
    collections.workflow(collectionId, { signal }),
    collections.statusHistory(collectionId, { signal })
  ])

  return {
    workflow: workflowResult.status === 'fulfilled' ? workflowResult.value : null,
    workflowError: workflowResult.status === 'rejected' ? 'The Temporal workflow history is not available.' : '',
    statusHistory: statusHistoryResult.status === 'fulfilled' ? statusHistoryResult.value : null,
    statusHistoryError: statusHistoryResult.status === 'rejected' ? 'The collection status history could not be loaded.' : ''
  }
}
