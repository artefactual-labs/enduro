import { defineComponent, h, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import { useCollectionWorkflow } from './useCollectionWorkflow'

const collection = ref({
  id: 77,
  workflowId: 'workflow-77',
  runId: 'run-77',
  status: 'in progress'
})
const collectionId = ref(77)
const collectionError = ref('')
const recentEvents = ref<Array<{
  sequence: number
  receivedAt: string
  type: string
  collectionId: number
}>>([])
const workflowData = ref()
const workflowError = ref<Error | null>(null)
const workflowLoading = ref(false)
const reloadWorkflowData = vi.fn()

vi.mock('~/loaders/collection-workflow', () => ({
  useCollectionWorkflowData: () => ({
    data: workflowData,
    error: workflowError,
    isLoading: workflowLoading,
    reload: reloadWorkflowData
  })
}))

mockNuxtImport('useCollectionDetails', () => () => ({
  collection,
  collectionId,
  errorMessage: collectionError
}))
mockNuxtImport('useEnduroMonitor', () => () => ({
  recentEvents,
  start: vi.fn()
}))

describe('useCollectionWorkflow', () => {
  beforeEach(() => {
    collectionId.value = 77
    recentEvents.value = []
    workflowData.value = undefined
    workflowError.value = null
    workflowLoading.value = false
    reloadWorkflowData.mockReset()
  })

  it('serializes lifecycle refreshes for matching SSE events', async () => {
    let resolveFirstReload!: () => void
    reloadWorkflowData
      .mockImplementationOnce(() => new Promise<void>((resolve) => {
        resolveFirstReload = resolve
      }))
      .mockResolvedValue(undefined)

    const Harness = defineComponent({
      setup() {
        useCollectionWorkflow()
        return () => h('div')
      }
    })

    const wrapper = await mountSuspended(Harness)
    const receivedAt = '2026-07-19T08:00:00.000Z'

    recentEvents.value = [{
      sequence: 1,
      receivedAt,
      type: 'collection:updated',
      collectionId: 77
    }]
    await vi.waitFor(() => expect(reloadWorkflowData).toHaveBeenCalledTimes(1))

    recentEvents.value = [{
      sequence: 2,
      receivedAt,
      type: 'collection:updated',
      collectionId: 77
    }, ...recentEvents.value]
    await nextTick()

    expect(reloadWorkflowData).toHaveBeenCalledTimes(1)

    resolveFirstReload()
    await vi.waitFor(() => expect(reloadWorkflowData).toHaveBeenCalledTimes(2))

    recentEvents.value = [{
      sequence: 3,
      receivedAt,
      type: 'collection:updated',
      collectionId: 88
    }, ...recentEvents.value]
    await nextTick()

    expect(reloadWorkflowData).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })
})
