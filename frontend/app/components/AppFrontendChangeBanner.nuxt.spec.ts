import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppFrontendChangeBanner from './AppFrontendChangeBanner.vue'

describe('AppFrontendChangeBanner', () => {
  it('renders the experimental UI announcement and docs link', async () => {
    const wrapper = await mountSuspended(AppFrontendChangeBanner)

    expect(wrapper.text()).toContain('Welcome to the experimental UI rewrite.')
    expect(wrapper.text()).toContain('The current UI is still available for now if you need it.')
    expect(wrapper.html()).toContain('#legacylisten-string')
    expect(wrapper.html()).toContain('text-center')
  })
})
