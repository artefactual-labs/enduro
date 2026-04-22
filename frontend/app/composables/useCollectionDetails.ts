import type { EnduroDetailedStoredCollection, EnduroStoredPipeline } from '~/openapi-generator'
import {
  EnduroDetailedStoredCollectionStatusEnum,
  RetryResultModeEnum
} from '~/openapi-generator'
import { useCollectionPageData } from '~/loaders/collection-details'

type CollectionAction = 'reload' | 'delete' | 'retry' | 'cancel' | 'decide-abandon' | 'decide-retry' | null

type CollectionDetailsState = {
  activeAction: CollectionAction
  actionErrorMessage: string
  retryModeMessage: string
}

const RUNNING_STATUSES = new Set<EnduroDetailedStoredCollectionStatusEnum>([
  EnduroDetailedStoredCollectionStatusEnum.New,
  EnduroDetailedStoredCollectionStatusEnum.InProgress,
  EnduroDetailedStoredCollectionStatusEnum.Queued,
  EnduroDetailedStoredCollectionStatusEnum.Pending
])

function createDefaultState(): CollectionDetailsState {
  return {
    activeAction: null,
    actionErrorMessage: '',
    retryModeMessage: ''
  }
}

function parseCollectionId(value: unknown): number {
  if (typeof value === 'string') {
    if (!/^\d+$/.test(value)) return 0
    const parsed = Number(value)
    return Number.isSafeInteger(parsed) ? parsed : 0
  }

  if (Array.isArray(value) && typeof value[0] === 'string') {
    if (!/^\d+$/.test(value[0])) return 0
    const parsed = Number(value[0])
    return Number.isSafeInteger(parsed) ? parsed : 0
  }

  return 0
}

export function useCollectionDetails() {
  const route = useRoute()
  const enduroApi = useEnduroApi()
  const monitor = useEnduroMonitor()
  const {
    data,
    error,
    isLoading,
    reload: reloadCollectionData
  } = useCollectionPageData()

  const collectionId = computed(() => parseCollectionId(route.params.id))
  const state = useState<CollectionDetailsState>('collection-details-state', createDefaultState)

  async function loadCollection(_force = false): Promise<void> {
    await reloadCollectionData()
  }

  async function runAction(
    action: Exclude<CollectionAction, null>,
    errorMessage: string,
    handler: () => Promise<void>
  ): Promise<boolean> {
    state.value.activeAction = action
    state.value.actionErrorMessage = ''

    try {
      await handler()
      return true
    } catch {
      state.value.actionErrorMessage = errorMessage
      return false
    } finally {
      state.value.activeAction = null
    }
  }

  async function reload(): Promise<void> {
    await runAction('reload', 'Could not refresh the collection.', () => loadCollection(true))
  }

  async function retry(): Promise<void> {
    const id = collection.value?.id
    if (!id) return

    const completed = await runAction('retry', 'Could not retry the collection.', async () => {
      const result = await enduroApi.collections.retry(id)
      state.value.retryModeMessage = result.mode === RetryResultModeEnum.ReconcileExistingAip
        ? 'Retry started in storage reconciliation mode.'
        : 'Retry started in full reprocess mode.'
      await loadCollection(true)
    })

    if (!completed) {
      state.value.retryModeMessage = ''
      return
    }
  }

  async function cancel(): Promise<void> {
    const id = collection.value?.id
    if (!id) return

    const completed = await runAction('cancel', 'Could not cancel the collection.', async () => {
      await enduroApi.collections.cancel(id)
      await loadCollection(true)
    })

    if (!completed) return
  }

  async function decide(option: 'ABANDON' | 'RETRY_ONCE'): Promise<void> {
    const id = collection.value?.id
    if (!id) return

    const action = option === 'ABANDON' ? 'decide-abandon' : 'decide-retry'
    const errorMessage = option === 'ABANDON'
      ? 'Could not abandon the collection workflow.'
      : 'Could not retry the collection workflow.'

    const completed = await runAction(action, errorMessage, async () => {
      await enduroApi.collections.decide(id, option)
      await loadCollection(true)
    })

    if (!completed) return
  }

  async function remove(): Promise<boolean> {
    const id = collection.value?.id
    if (!id) return false

    const completed = await runAction('delete', 'Could not delete the collection.', async () => {
      await enduroApi.collections.remove(id)
    })

    if (!completed) return false

    state.value = createDefaultState()
    await navigateTo('/collections')
    return true
  }

  function dismissRetryModeMessage(): void {
    state.value.retryModeMessage = ''
  }

  watch(collectionId, () => {
    state.value.activeAction = null
    state.value.actionErrorMessage = ''
    state.value.retryModeMessage = ''
  })

  const collection = computed<EnduroDetailedStoredCollection | null>(() => data.value?.collection ?? null)
  const pipeline = computed<EnduroStoredPipeline | null>(() => data.value?.pipeline ?? null)
  const errorMessage = computed(() => error.value?.message ?? '')
  const actionErrorMessage = computed(() => state.value.actionErrorMessage)
  const hasLoaded = computed(() => data.value !== undefined || error.value !== null)
  const activeAction = computed(() => state.value.activeAction)
  const retryModeMessage = computed(() => state.value.retryModeMessage)
  const collectionName = computed(() => collection.value?.name || `Collection #${collectionId.value}`)
  const isPending = computed(() => collection.value?.status === EnduroDetailedStoredCollectionStatusEnum.Pending)
  const isRunning = computed(() => {
    const status = collection.value?.status
    return status ? RUNNING_STATUSES.has(status) : false
  })
  const canDelete = computed(() => Boolean(collection.value) && !isRunning.value)
  const canRetry = computed(() => collection.value?.status === EnduroDetailedStoredCollectionStatusEnum.Error)
  const canCancel = computed(() => collection.value?.status === EnduroDetailedStoredCollectionStatusEnum.InProgress)

  watch(() => monitor.recentEvents.value[0]?.receivedAt, () => {
    const latest = monitor.recentEvents.value[0]
    if (!latest || latest.type !== 'collection:updated') return
    if (latest.collectionId !== collectionId.value) return
    void loadCollection(true)
  })

  return {
    activeAction,
    actionErrorMessage,
    canCancel,
    canDelete,
    canRetry,
    collection,
    collectionId,
    collectionName,
    errorMessage,
    hasLoaded,
    isLoading,
    isPending,
    isRunning,
    pipeline,
    retryModeMessage,
    cancel,
    decide,
    dismissRetryModeMessage,
    loadCollection,
    reload,
    remove,
    retry
  }
}
