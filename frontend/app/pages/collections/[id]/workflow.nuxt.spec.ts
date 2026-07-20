import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockComponent, mockNuxtImport } from '@nuxt/test-utils/runtime'
import {
  type EnduroCollectionStatusHistory,
  EnduroCollectionStatusHistoryAvailabilityEnum as Availability,
  EnduroCollectionStatusTransitionPreviousStatusEnum as PreviousStatus,
  EnduroCollectionStatusTransitionStatusEnum as Status
} from '~/openapi-generator'

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
      runId: 'run-77',
      status: 'done'
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
    statusHistory: ref<EnduroCollectionStatusHistory>({
      availability: Availability.Unavailable,
      transitions: []
    }),
    statusHistoryErrorMessage: ref(''),
    workflowErrorMessage: ref(''),
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

  it('labels the workflow start time explicitly', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    const labels = wrapper.findAll('dt').map(label => label.text())
    expect(labels).toContain('Workflow started')
    expect(labels).not.toContain('Processing started')
  })

  it('renders workflow completion metadata and duration', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    const labels = wrapper.findAll('dt').map(label => label.text())
    expect(labels).toContain('Completed')
    expect(wrapper.text()).toContain('COMPLETED')
    expect(wrapper.text()).toContain('took 5m')
  })

  it('describes the Temporal-derived workflow sections', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Activity summary')
    expect(wrapper.text()).toContain('activity executions derived from the Temporal workflow event history')
    expect(wrapper.text()).toContain('Workflow event history')
    expect(wrapper.text()).toContain('Temporal events recorded for this workflow execution, shown newest first')
    wrapper.unmount()
  })

  it('shows a safe empty state for legacy collections', async () => {
    useCollectionWorkflowMock.mockReturnValue(createWorkflowState())

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Status history is not available for this workflow run')
    expect(wrapper.text()).toContain('started or retried after the upgrade')
  })

  it('keeps collection lifecycle visible when Temporal is unavailable', async () => {
    const state = createWorkflowState()
    state.workflowErrorMessage.value = 'The Temporal workflow history is not available.'
    useCollectionWorkflowMock.mockReturnValue(state)

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Temporal workflow unavailable')
    expect(wrapper.text()).toContain('Collection lifecycle')
    expect(wrapper.text()).toContain('Status history is not available for this workflow run')
    expect(wrapper.text()).not.toContain('Activity summary')
  })

  it('renders recorded collection status periods', async () => {
    const state = createWorkflowState()
    state.statusHistory.value = {
      availability: Availability.Available,
      transitions: [
        {
          id: 1,
          isRunStart: true,
          occurredAt: new Date('2026-04-22T18:40:00Z'),
          reason: 'collection_created',
          runId: 'run-77',
          status: Status.Queued,
          workflowId: 'workflow-77'
        },
        {
          id: 2,
          isRunStart: false,
          occurredAt: new Date('2026-04-22T18:47:00Z'),
          previousStatus: PreviousStatus.Queued,
          reason: 'pipeline_acquired',
          runId: 'run-77',
          status: Status.InProgress,
          workflowId: 'workflow-77'
        }
      ]
    }
    useCollectionWorkflowMock.mockReturnValue(state)

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Queued')
    expect(wrapper.text()).toContain('In progress')
    expect(wrapper.text()).toContain('Pipeline acquired')
    expect(wrapper.text()).toContain('Time by status')

    let transitionHistoryButton = wrapper.findAll('button')
      .find(button => button.text() === 'Show exact transition history')
    expect(transitionHistoryButton?.attributes('aria-pressed')).toBe('false')

    await transitionHistoryButton?.trigger('click')
    transitionHistoryButton = wrapper.findAll('button')
      .find(button => button.text() === 'Hide exact transition history')

    expect(transitionHistoryButton?.attributes('aria-pressed')).toBe('true')
    expect(wrapper.text()).toContain('Time in status')
    wrapper.unmount()
  })

  it('renders recorded transitions from an incomplete history', async () => {
    const state = createWorkflowState()
    state.statusHistory.value = {
      availability: Availability.Partial,
      transitions: [
        {
          id: 2,
          isRunStart: false,
          occurredAt: new Date('2026-04-22T18:47:00Z'),
          previousStatus: PreviousStatus.Queued,
          reason: 'pipeline_acquired',
          runId: 'run-77',
          status: Status.InProgress,
          workflowId: 'workflow-77'
        }
      ]
    }
    useCollectionWorkflowMock.mockReturnValue(state)

    const wrapper = await mountSuspended(WorkflowPage, {
      route: '/collections/77/workflow'
    })

    expect(wrapper.text()).toContain('Status history is incomplete')
    expect(wrapper.text()).toContain('Earlier statuses and their durations are not included')
    expect(wrapper.text()).toContain('In progress')
    expect(wrapper.text()).toContain('Pipeline acquired')
    expect(wrapper.text()).toContain('Recorded time by status')
    expect(wrapper.text()).not.toContain('Status history is not available for this workflow run')
    wrapper.unmount()
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
