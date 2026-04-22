import { nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockComponent, mockNuxtImport } from '@nuxt/test-utils/runtime'

import CollectionsPage from './index.vue'

const { useCollectionsBrowserMock, useDashboardUiOptionsMock } = vi.hoisted(() => ({
  useCollectionsBrowserMock: vi.fn(),
  useDashboardUiOptionsMock: vi.fn()
}))

mockNuxtImport('useCollectionsBrowser', () => useCollectionsBrowserMock)
mockNuxtImport('useDashboardUiOptions', () => useDashboardUiOptionsMock)

mockComponent('UTooltip', async () => {
  const { defineComponent, h } = await import('vue')

  return defineComponent({
    setup(_, { slots }) {
      return () => h('span', slots.default?.())
    }
  })
})

function createCollectionsBrowserState(overrides: Partial<ReturnType<typeof useCollectionsBrowserMock>> = {}) {
  return {
    statusOptions: [{ label: 'Status', value: 'all' }],
    dateOptions: [{ label: 'Creation date', value: 'all' }],
    fieldOptions: [{ label: 'Name', value: 'name' }],
    selectedStatus: ref('all'),
    selectedDate: ref('all'),
    selectedField: ref('name'),
    query: ref(''),
    isLoading: ref(false),
    hasError: ref(false),
    validQuery: ref(null),
    rows: ref([
      {
        id: 4,
        name: 'transfer2.zip',
        startedAt: '2026-04-22T19:55:00Z',
        completedAt: '2026-04-22T12:55:00Z',
        status: 'done'
      }
    ]),
    queryHelp: ref('Prefix and case-insensitive name matching, e.g. "DPJ-SIP-97".'),
    queryError: ref(''),
    hasRows: ref(true),
    canGoPrev: ref(false),
    canGoNext: ref(false),
    showPager: ref(false),
    statusColor: vi.fn(() => 'success'),
    formatDateTime: vi.fn((value: string) => value),
    onSubmit: vi.fn(),
    onReset: vi.fn(),
    onRetry: vi.fn(),
    onGoHome: vi.fn(),
    onGoPrev: vi.fn(),
    onGoNext: vi.fn(),
    onCollectionRowSelect: vi.fn(),
    ...overrides
  }
}

function createDashboardUiOptionsState(collectionsSearchOpen = ref(false)) {
  return {
    collectionsSearchOpen,
    setCollectionsSearchOpen: vi.fn((value: boolean) => {
      collectionsSearchOpen.value = value
    })
  }
}

describe('collections index page', () => {
  beforeEach(() => {
    useCollectionsBrowserMock.mockReset()
    useDashboardUiOptionsMock.mockReset()
  })

  it('renders the page heading, actions, and collapsed search toggle', async () => {
    useCollectionsBrowserMock.mockReturnValue(createCollectionsBrowserState())
    useDashboardUiOptionsMock.mockReturnValue(createDashboardUiOptionsState())

    const wrapper = await mountSuspended(CollectionsPage, {
      route: '/collections'
    })

    expect(wrapper.text()).toContain('Collections')
    expect(wrapper.text()).toContain('Batch import')
    expect(wrapper.text()).toContain('Bulk operation')
    expect(wrapper.text()).toContain('Search')
    expect(wrapper.text()).not.toContain('Search query')
    expect(wrapper.text()).toContain('transfer2.zip')
  })

  it('opens the search panel from the toolbar toggle', async () => {
    useCollectionsBrowserMock.mockReturnValue(createCollectionsBrowserState())
    const uiOptions = createDashboardUiOptionsState()
    useDashboardUiOptionsMock.mockReturnValue(uiOptions)

    const wrapper = await mountSuspended(CollectionsPage, {
      route: '/collections'
    })

    const searchButton = wrapper.findAll('button').find(node => node.text() === 'Search')
    expect(searchButton).toBeTruthy()
    await searchButton?.trigger('click')
    await nextTick()

    expect(uiOptions.setCollectionsSearchOpen).toHaveBeenCalledWith(true)
    expect(wrapper.text()).toContain('Search query')
  })

  it('opens search automatically when filters are active', async () => {
    useCollectionsBrowserMock.mockReturnValue(createCollectionsBrowserState({
      query: ref('DPJ-SIP')
    }))
    const uiOptions = createDashboardUiOptionsState()
    useDashboardUiOptionsMock.mockReturnValue(uiOptions)

    const wrapper = await mountSuspended(CollectionsPage, {
      route: '/collections?q=DPJ-SIP'
    })
    await nextTick()

    expect(uiOptions.setCollectionsSearchOpen).toHaveBeenCalledWith(true, { persist: false })
    expect(wrapper.text()).toContain('Search query')
  })

  it('shows the warning alert and retries when the search fails', async () => {
    const state = createCollectionsBrowserState({
      hasError: ref(true),
      isLoading: ref(true),
      rows: ref([]),
      hasRows: ref(false)
    })
    useCollectionsBrowserMock.mockReturnValue(state)
    useDashboardUiOptionsMock.mockReturnValue(createDashboardUiOptionsState())

    const wrapper = await mountSuspended(CollectionsPage, {
      route: '/collections'
    })

    expect(wrapper.text()).toContain('Search error')
    expect(wrapper.text()).not.toContain('No results')
    expect(wrapper.text()).not.toContain('Batch import')
    expect(wrapper.text()).not.toContain('Bulk operation')

    const retryButton = wrapper.findAll('button').find(node => node.text() === 'Retry')
    expect(retryButton).toBeTruthy()
    const searchButton = wrapper.findAll('button').find(node => node.text() === 'Search')
    expect(searchButton).toBeFalsy()
  })
})
