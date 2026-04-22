import type { EnduroCollectionWorkflowStatus } from '~/openapi-generator'
import { useCollectionWorkflowData } from '~/loaders/collection-workflow'
import { parseWorkflowStatus } from '~/utils/workflow-history'

export function useCollectionWorkflow() {
  const {
    collection,
    errorMessage: collectionErrorMessage
  } = useCollectionDetails()
  const {
    data,
    error,
    isLoading,
    reload
  } = useCollectionWorkflowData()

  async function loadWorkflow(_force = false): Promise<void> {
    await reload()
  }

  watch(
    () => {
      const value = collection.value
      if (!value) return ''
      return [
        value.id,
        value.workflowId ?? '',
        value.runId ?? '',
        value.status
      ].join('|')
    },
    (nextSignature, previousSignature) => {
      if (!nextSignature || nextSignature === previousSignature) return
      void loadWorkflow(true)
    }
  )

  const parsedWorkflow = computed(() => parseWorkflowStatus(data.value?.workflow as EnduroCollectionWorkflowStatus | null))
  const errorMessage = computed(() => error.value?.message || collectionErrorMessage.value)
  const hasLoaded = computed(() => data.value !== undefined || error.value !== null)

  return {
    collection,
    errorMessage,
    hasLoaded,
    isLoading,
    parsedWorkflow,
    loadWorkflow
  }
}
