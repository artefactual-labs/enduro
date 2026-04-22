import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import PipelinesPage from './index.vue'

const { usePipelinesMock } = vi.hoisted(() => ({
  usePipelinesMock: vi.fn()
}))

mockNuxtImport('usePipelines', () => usePipelinesMock)

function createPipelinesState(overrides: Partial<ReturnType<typeof usePipelinesMock>> = {}) {
  return {
    errorMessage: ref(''),
    hasLoaded: ref(true),
    isLoading: ref(false),
    loadPipelines: vi.fn(),
    pipelines: ref([
      {
        id: '367424b6-101c-49f1-b6ef-2653ddda00eb',
        name: 'am',
        current: 0,
        capacity: 3,
        status: 'active'
      }
    ]),
    ...overrides
  }
}

describe('pipelines index page', () => {
  beforeEach(() => {
    usePipelinesMock.mockReset()
  })

  it('shows the error state without also rendering the empty state', async () => {
    usePipelinesMock.mockReturnValue(createPipelinesState({
      errorMessage: ref('Could not load pipelines.'),
      hasLoaded: ref(false),
      pipelines: ref([])
    }))

    const wrapper = await mountSuspended(PipelinesPage, {
      route: '/pipelines'
    })

    expect(wrapper.text()).toContain('Pipeline status unavailable')
    expect(wrapper.text()).not.toContain('No pipelines')
  })
})
