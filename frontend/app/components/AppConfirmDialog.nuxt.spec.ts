import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppConfirmDialog from './AppConfirmDialog.vue'

describe('AppConfirmDialog', () => {
  it('passes the compact width class to the modal content', async () => {
    const wrapper = await mountSuspended(AppConfirmDialog, {
      props: {
        open: true,
        title: 'Delete collection'
      }
    })

    const modal = wrapper.findComponent({ name: 'UModal' })
    expect(modal.props('class')).toBe('max-w-md')
  })

  it('blocks close requests while a confirm action is pending', async () => {
    const wrapper = await mountSuspended(AppConfirmDialog, {
      props: {
        open: true,
        pending: true,
        title: 'Delete collection'
      }
    })

    const modal = wrapper.findComponent({ name: 'UModal' })
    modal.vm.$emit('update:open', false)

    expect(wrapper.emitted('update:open')).toBeUndefined()
  })
})
