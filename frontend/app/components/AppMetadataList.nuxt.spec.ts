import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppMetadataList from './AppMetadataList.vue'

describe('AppMetadataList', () => {
  it('renders split rows and hides items without a value or slot', async () => {
    const wrapper = await mountSuspended(AppMetadataList, {
      props: {
        items: [
          { key: 'status', label: 'Status', value: 'DONE' },
          { key: 'empty', label: 'Empty', value: undefined },
          { key: 'pipeline', label: 'Pipeline', slot: 'pipeline' }
        ]
      },
      slots: {
        pipeline: () => 'Archivematica'
      }
    })

    expect(wrapper.text()).toContain('Status')
    expect(wrapper.text()).toContain('DONE')
    expect(wrapper.text()).toContain('Pipeline')
    expect(wrapper.text()).toContain('Archivematica')
    expect(wrapper.text()).not.toContain('Empty')
  })

  it('renders stacked labels in the compact layout', async () => {
    const wrapper = await mountSuspended(AppMetadataList, {
      props: {
        layout: 'stacked',
        items: [{ key: 'id', label: 'Identifier', value: '123' }]
      }
    })

    expect(wrapper.text()).toContain('Identifier')
    expect(wrapper.text()).toContain('123')
    expect(wrapper.html()).toContain('uppercase')
  })
})
