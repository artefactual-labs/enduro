import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppHeader from './AppHeader.vue'

describe('AppHeader', () => {
  it('renders navigation and the logo in the header shell', async () => {
    const wrapper = await mountSuspended(AppHeader, {
      route: '/pipelines/367424b6-101c-49f1-b6ef-2653ddda00eb'
    })

    expect(wrapper.text()).toContain('Enduro')
    expect(wrapper.text()).toContain('Collections')
    expect(wrapper.text()).toContain('Pipelines')
  })
})
