import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockComponent, mountSuspended } from '@nuxt/test-utils/runtime'
import PipelinePage from './[id].vue'

const { usePipelinePageDataMock } = vi.hoisted(() => ({
  usePipelinePageDataMock: vi.fn()
}))

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

mockComponent('AppPipelineCapacity', async () => {
  const { defineComponent, h } = await import('vue')

  return defineComponent({
    props: {
      current: {
        type: Number,
        default: 0
      },
      capacity: {
        type: Number,
        default: 0
      }
    },
    setup(props) {
      return () => h('span', `${props.current} / ${props.capacity}`)
    }
  })
})

vi.mock('~/loaders/pipeline-details', () => ({
  usePipelinePageData: usePipelinePageDataMock
}))

function createPipelineLoaderState(overrides: Partial<ReturnType<typeof usePipelinePageDataMock>> = {}) {
  return {
    data: ref({
      pipeline: {
        id: '367424b6-101c-49f1-b6ef-2653ddda00eb',
        name: 'am',
        status: 'active',
        current: 0,
        capacity: 3
      },
      pipelineId: '367424b6-101c-49f1-b6ef-2653ddda00eb',
      processingConfigurations: ['automated', 'default']
    }),
    error: ref(null),
    isLoading: ref(false),
    reload: vi.fn(),
    ...overrides
  }
}

describe('pipeline detail page', () => {
  beforeEach(() => {
    usePipelinePageDataMock.mockReset()
  })

  it('renders pipeline metadata and Archivematica processing configurations', async () => {
    usePipelinePageDataMock.mockReturnValue(createPipelineLoaderState())

    const wrapper = await mountSuspended(PipelinePage, {
      route: '/pipelines/367424b6-101c-49f1-b6ef-2653ddda00eb'
    })

    expect(wrapper.text()).toContain('Pipeline')
    expect(wrapper.text()).toContain('am')
    expect(wrapper.text()).toContain('Identifier')
    expect(wrapper.text()).toContain('Processing configurations')
    expect(wrapper.text()).toContain('reported by Archivematica')
    expect(wrapper.text()).toContain('automated')
    expect(wrapper.text()).toContain('default')
  })

  it('does not show the pipeline id in the breadcrumb while the loader is still pending', async () => {
    usePipelinePageDataMock.mockReturnValue(createPipelineLoaderState({
      data: ref(null),
      error: ref(null),
      isLoading: ref(true)
    }))

    const wrapper = await mountSuspended(PipelinePage, {
      route: '/pipelines/367424b6-101c-49f1-b6ef-2653ddda00eb'
    })

    expect(wrapper.text()).toContain('Pipelines')
    expect(wrapper.text()).not.toContain('367424b6-101c-49f1-b6ef-2653ddda00eb')
  })
})
