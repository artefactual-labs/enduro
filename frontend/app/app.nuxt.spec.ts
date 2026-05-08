import { describe, expect, it } from 'vitest'
import { mockComponent, mountSuspended } from '@nuxt/test-utils/runtime'

import App from './app.vue'

mockComponent('AppFrontendChangeBanner', async () => {
  const { defineComponent, h } = await import('vue')
  return defineComponent({
    setup() {
      return () => h('div', 'Banner stub')
    }
  })
})

mockComponent('AppHeader', async () => {
  const { defineComponent, h } = await import('vue')
  return defineComponent({
    setup() {
      return () => h('header', 'Header stub')
    }
  })
})

mockComponent('AppFooter', async () => {
  const { defineComponent, h } = await import('vue')
  return defineComponent({
    setup() {
      return () => h('footer', 'Footer stub')
    }
  })
})

mockComponent('NuxtPage', async () => {
  const { defineComponent, h } = await import('vue')
  return defineComponent({
    setup() {
      return () => h('div', 'Page outlet stub')
    }
  })
})

describe('App shell', () => {
  it('renders the banner, header, page outlet, and footer', async () => {
    const wrapper = await mountSuspended(App, {
      route: '/collections'
    })

    expect(wrapper.text()).toContain('Banner stub')
    expect(wrapper.text()).toContain('Header stub')
    expect(wrapper.text()).toContain('Page outlet stub')
    expect(wrapper.text()).toContain('Footer stub')
  })
})
