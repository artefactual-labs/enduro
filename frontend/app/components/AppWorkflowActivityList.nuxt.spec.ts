import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppWorkflowActivityList from './AppWorkflowActivityList.vue'

describe('AppWorkflowActivityList', () => {
  it('renders workflow activities in a condensed table layout', async () => {
    const wrapper = await mountSuspended(AppWorkflowActivityList, {
      props: {
        activities: [
          {
            id: 12,
            attempts: 3,
            completedAt: '2026-04-22T18:47:12.000Z',
            details: 'User selection: approve.',
            durationSeconds: '7',
            isLocal: false,
            name: 'async-completion-activity',
            replayedAt: null,
            startedAt: '2026-04-22T18:47:05.000Z',
            status: 'done'
          },
          {
            id: 17,
            attempts: 1,
            completedAt: null,
            details: '',
            durationSeconds: null,
            isLocal: true,
            name: 'updatePackageLocalActivity',
            replayedAt: '2026-04-22T18:47:00.000Z',
            startedAt: null,
            status: null
          }
        ],
        formatDateTime: (value: string | null) => value ?? 'N/A',
        statusColor: () => 'success'
      }
    })

    expect(wrapper.text()).toContain('Activity')
    expect(wrapper.text()).toContain('ID')
    expect(wrapper.text()).toContain('Status')
    expect(wrapper.text()).toContain('Time')
    expect(wrapper.text()).toContain('Duration')
    expect(wrapper.text()).toContain('Attempts')
    expect(wrapper.text()).toContain('Details')
    expect(wrapper.text()).toContain('async-completion-activity')
    expect(wrapper.text()).toContain('DONE')
    expect(wrapper.text()).toContain('Started')
    expect(wrapper.text()).toContain('7s')
    expect(wrapper.text()).toContain('3')
    expect(wrapper.text()).toContain('User selection: approve.')
    expect(wrapper.text()).toContain('updatePackageLocalActivity')
    expect(wrapper.text()).toContain('Local')
    expect(wrapper.text()).toContain('Replayed')
    expect(wrapper.text()).toContain('#12')
    expect(wrapper.text()).toContain('#17')
  })
})
