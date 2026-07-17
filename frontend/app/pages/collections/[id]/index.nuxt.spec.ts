import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockComponent, mockNuxtImport } from '@nuxt/test-utils/runtime'

import CollectionOverviewPage from './index.vue'

const { useCollectionDetailsMock, useCollectionsListLocationMock } = vi.hoisted(() => ({
  useCollectionDetailsMock: vi.fn(),
  useCollectionsListLocationMock: vi.fn()
}))

mockNuxtImport('useCollectionDetails', () => useCollectionDetailsMock)
mockNuxtImport('useCollectionsListLocation', () => useCollectionsListLocationMock)

mockComponent('AppUuid', async () => {
  const { defineComponent, h } = await import('vue')

  return defineComponent({
    props: {
      value: {
        type: String,
        default: ''
      }
    },
    setup(props) {
      return () => h('span', props.value)
    }
  })
})

function createCollectionDetailsState(collectionOverrides: { startedAt?: Date } = {}) {
  return {
    activeAction: ref(null),
    actionErrorMessage: ref(''),
    canCancel: ref(false),
    canDelete: ref(false),
    canRetry: ref(false),
    collection: ref({
      id: 77,
      createdAt: new Date('2026-04-22T18:45:00.000Z'),
      startedAt: new Date('2026-04-22T18:47:00.000Z'),
      status: 'in progress',
      workflowId: 'workflow-77',
      runId: 'run-77',
      ...collectionOverrides
    }),
    errorMessage: ref(''),
    hasLoaded: ref(true),
    isLoading: ref(false),
    isPending: ref(false),
    pipeline: ref(null),
    retryModeMessage: ref(''),
    cancel: vi.fn(),
    decide: vi.fn(),
    dismissRetryModeMessage: vi.fn(),
    reload: vi.fn(),
    remove: vi.fn(),
    retry: vi.fn()
  }
}

describe('collection overview page', () => {
  beforeEach(() => {
    useCollectionDetailsMock.mockReset()
    useCollectionDetailsMock.mockReturnValue(createCollectionDetailsState())
    useCollectionsListLocationMock.mockReset()
    useCollectionsListLocationMock.mockReturnValue(ref({ path: '/collections', query: {} }))
  })

  it('identifies the processing start time explicitly', async () => {
    const wrapper = await mountSuspended(CollectionOverviewPage, {
      route: '/collections/77'
    })

    const labels = wrapper.findAll('dt').map(label => label.text())
    expect(labels).toContain('Created')
    expect(labels).toContain('Processing started')
    expect(labels).not.toContain('Workflow started')
  })

  it('explains when collection processing has not started', async () => {
    useCollectionDetailsMock.mockReturnValue(createCollectionDetailsState({ startedAt: undefined }))

    const wrapper = await mountSuspended(CollectionOverviewPage, {
      route: '/collections/77'
    })

    const labels = wrapper.findAll('dt').map(label => label.text())
    expect(labels).toContain('Processing started')
    expect(wrapper.text()).toContain('Processing has not started yet.')
  })
})
