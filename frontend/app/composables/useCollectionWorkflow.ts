import type {
  EnduroCollectionStatusHistory,
  EnduroCollectionWorkflowStatus
} from '~/openapi-generator'
import { useCollectionWorkflowData } from '~/loaders/collection-workflow'
import { parseWorkflowStatus } from '~/utils/workflow-history'

export function useCollectionWorkflow() {
  const {
    collection,
    collectionId,
    errorMessage: collectionErrorMessage
  } = useCollectionDetails()
  const monitor = useEnduroMonitor()
  const {
    data,
    error,
    isLoading,
    reload
  } = useCollectionWorkflowData()

  let reloadPending = false
  let reloadPromise: Promise<void> | null = null

  function loadWorkflow(_force = false): Promise<void> {
    reloadPending = true
    if (!reloadPromise) {
      reloadPromise = (async () => {
        try {
          while (reloadPending) {
            reloadPending = false
            await reload()
          }
        } finally {
          reloadPromise = null
        }
      })()
    }

    return reloadPromise
  }

  watch(() => monitor.recentEvents.value[0]?.sequence, () => {
    const latest = monitor.recentEvents.value[0]
    if (!latest || latest.type !== 'collection:updated') return
    if (latest.collectionId !== collectionId.value) return
    void loadWorkflow(true)
  })

  const parsedWorkflow = computed(() => parseWorkflowStatus(data.value?.workflow as EnduroCollectionWorkflowStatus | null))
  const errorMessage = computed(() => error.value?.message || collectionErrorMessage.value)
  const workflowErrorMessage = computed(() => data.value?.workflowError ?? '')
  const statusHistory = computed<EnduroCollectionStatusHistory | null>(() => data.value?.statusHistory ?? null)
  const statusHistoryErrorMessage = computed(() => data.value?.statusHistoryError ?? '')
  const hasLoaded = computed(() => data.value !== undefined || error.value !== null)

  return {
    collection,
    errorMessage,
    hasLoaded,
    isLoading,
    parsedWorkflow,
    statusHistory,
    statusHistoryErrorMessage,
    workflowErrorMessage,
    loadWorkflow
  }
}
