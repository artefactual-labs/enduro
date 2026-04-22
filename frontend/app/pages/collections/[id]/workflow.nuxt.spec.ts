import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockComponent, mockNuxtImport } from '@nuxt/test-utils/runtime'

import WorkflowPage from './workflow.vue'

const { useCollectionWorkflowMock, useCollectionsListLocationMock } = vi.hoisted(() => ({
  useCollectionWorkflowMock: vi.fn(),
  useCollectionsListLocationMock: vi.fn()
}))

mockNuxtImport('useCollectionWorkflow', () => useCollectionWorkflowMock)
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

mockComponent('AppWorkflowActivityList', async () => {
  const { defineComponent, h } = await import('vue')

  return defineComponent({
    props: {
      activities: {
        type: Array,
        default: () => []
      }
    },
    setup(props) {
      return () => h('div', JSON.stringify(props.activities))
    }
  })
})

function createWorkflowState(status = 'completed') {
  return {
    collection: ref({
      id: 77,
      workflowId: 'workflow-77',
      runId: 'run-77'
    }),
    errorMessage: ref(''),
    hasLoaded: ref(true),
    isLoading: ref(false),
    parsedWorkflow: ref({
      status,
      startedAt: '2026-04-22T18:47:00.000Z',
      completedAt: '2026-04-22T18:52:00.000Z',
      workflowError: '',
      activityError: '',
      activities: [],
      events: [
        {
          id: 17,
          type: 'ActivityTaskFailed',
          activityName: 'store',
          description: '{\n  "message": "failure"\n}',
          eventTime: '2026-04-22T18:48:00.000Z'
        }
      ]
    }),
    loadWorkflow: vi.fn()
  }
}

describe('workflow page', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  beforeEach(() => {
    useCollectionsListLocationMock.mockReset()
    useCollectionsListLocationMock.mockReturnValue(ref({ path: '/collections', query: {} }))
    useCollectionWorkflowMock.mockReset()
  })

  it('renders started and completed metadata when timestamps are present', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Started')
    expect(wrapper.text()).toContain('Completed')
    expect(wrapper.text()).toContain('COMPLETED')
    expect(wrapper.text()).toContain('took 5m')
  })

  it('keeps the selected history event content mounted while the modal starts closing', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    const detailsButton = wrapper.findAll('button').find(node => node.text() === 'View details')
    expect(detailsButton).toBeTruthy()

    await detailsButton!.trigger('click')
    expect(document.body.textContent ?? '').toContain('failure')

    const modal = wrapper.findComponent({ name: 'UModal' })
    modal.vm.$emit('update:open', false)

    expect(document.body.textContent ?? '').toContain('failure')
  })

  it('does not poll while the workflow is already terminal', async () => {
    vi.useFakeTimers()

    const state = createWorkflowState('completed')
    useCollectionWorkflowMock.mockReturnValue(state)

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    await vi.advanceTimersByTimeAsync(15_000)

    expect(state.loadWorkflow).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('polls while the workflow is still running', async () => {
    vi.useFakeTimers()

    const state = createWorkflowState('running')
    useCollectionWorkflowMock.mockReturnValue(state)

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    await vi.advanceTimersByTimeAsync(5_000)

    expect(state.loadWorkflow).toHaveBeenCalledTimes(1)
    expect(state.loadWorkflow).toHaveBeenCalledWith(true)
    wrapper.unmount()
  })
})
