import { defineComponent, h, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import { RetryResultModeEnum } from '~/openapi-generator'
import { useCollectionDetails } from './useCollectionDetails'

const { navigateToMock } = vi.hoisted(() => ({
  navigateToMock: vi.fn()
}))

const route = ref({ params: { id: '55' } })
const recentEvents = ref<Array<{ receivedAt: string, type: string, collectionId: number }>>([])
const collectionData = ref({
  collection: {
    id: 77,
    name: 'Example collection',
    status: 'error'
  },
  pipeline: null
})
const loaderError = ref<Error | null>(null)
const isLoading = ref(false)
const reloadCollectionData = vi.fn()

const retrySpy = vi.fn(async () => ({ mode: RetryResultModeEnum.ReconcileExistingAip }))
const cancelSpy = vi.fn(async () => {})
const decideSpy = vi.fn(async () => {})
const removeSpy = vi.fn(async () => {})

vi.mock('~/loaders/collection-details', () => ({
  useCollectionPageData: () => ({
    data: collectionData,
    error: loaderError,
    isLoading,
    reload: reloadCollectionData
  })
}))

mockNuxtImport('useRoute', () => () => route.value)
mockNuxtImport('useEnduroApi', () => () => ({
  collections: {
    retry: retrySpy,
    cancel: cancelSpy,
    decide: decideSpy,
    remove: removeSpy
  }
}))
mockNuxtImport('useEnduroMonitor', () => () => ({
  recentEvents
}))
mockNuxtImport('navigateTo', () => navigateToMock)

describe('useCollectionDetails', () => {
  beforeEach(() => {
    route.value = { params: { id: '55' } }
    recentEvents.value = []
    collectionData.value = {
      collection: {
        id: 77,
        name: 'Example collection',
        status: 'error'
      },
      pipeline: null
    }
    loaderError.value = null
    isLoading.value = false

    navigateToMock.mockReset()
    reloadCollectionData.mockReset()
    retrySpy.mockClear()
    retrySpy.mockResolvedValue({ mode: RetryResultModeEnum.ReconcileExistingAip })
    cancelSpy.mockClear()
    decideSpy.mockClear()
    removeSpy.mockClear()
  })

  it('uses the loaded collection id for detail-page actions', async () => {
    let details!: ReturnType<typeof useCollectionDetails>

    const Harness = defineComponent({
      setup() {
        details = useCollectionDetails()
        return () => h('div')
      }
    })

    await mountSuspended(Harness)

    await details.retry()

    collectionData.value = {
      collection: {
        id: 77,
        name: 'Example collection',
        status: 'in progress'
      },
      pipeline: null
    }
    await details.cancel()
    await details.decide('ABANDON')
    await details.remove()

    expect(retrySpy).toHaveBeenCalledWith(77)
    expect(cancelSpy).toHaveBeenCalledWith(77)
    expect(decideSpy).toHaveBeenCalledWith(77, 'ABANDON')
    expect(removeSpy).toHaveBeenCalledWith(77)
    expect(navigateToMock).toHaveBeenCalledWith('/collections')
    expect(reloadCollectionData).toHaveBeenCalledTimes(3)
    expect(details.retryModeMessage.value).toBe('')
  })
})
