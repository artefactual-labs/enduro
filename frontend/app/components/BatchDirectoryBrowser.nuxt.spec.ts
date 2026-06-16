import { nextTick } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import BatchDirectoryBrowser from './BatchDirectoryBrowser.vue'

const { browseMock } = vi.hoisted(() => ({
  browseMock: vi.fn()
}))

mockNuxtImport('useEnduroApi', () => () => ({
  batches: {
    browse: browseMock
  }
}))

describe('BatchDirectoryBrowser', () => {
  beforeEach(() => {
    browseMock.mockReset()
    browseMock.mockResolvedValue({
      absolutePath: '/batch-root',
      entries: [
        {
          absolutePath: '/batch-root/alpha',
          name: 'alpha',
          path: 'alpha'
        }
      ],
      path: '',
      truncated: false
    })
  })

  it('loads the browser root when opened', async () => {
    const wrapper = await mountSuspended(BatchDirectoryBrowser, {
      props: {
        open: true
      }
    })
    await flushPromises()

    expect(browseMock).toHaveBeenCalledWith({})
    expect(document.body.textContent).toContain('Browse source directory')
    expect(document.body.textContent).toContain('No directory selected.')
    expect(document.body.textContent).not.toContain('/batch-root')

    const tree = wrapper.findComponent({ name: 'UTree' })
    expect(tree.props('items')[0].path).toBe('alpha')
    expect(tree.props('modelValue')).toBeUndefined()

    const useButton = wrapper.findAllComponents({ name: 'UButton' })
      .find(button => button.props('label') === 'Use directory')
    expect(useButton?.props('disabled')).toBe(true)
  })

  it('loads child directories lazily when expanded', async () => {
    browseMock
      .mockResolvedValueOnce({
        absolutePath: '/batch-root',
        entries: [
          {
            absolutePath: '/batch-root/alpha',
            name: 'alpha',
            path: 'alpha'
          }
        ],
        path: '',
        truncated: false
      })
      .mockResolvedValueOnce({
        absolutePath: '/batch-root/alpha',
        entries: [
          {
            absolutePath: '/batch-root/alpha/child',
            name: 'child',
            path: 'alpha/child'
          }
        ],
        path: 'alpha',
        truncated: false
      })

    const wrapper = await mountSuspended(BatchDirectoryBrowser, {
      props: {
        open: true
      }
    })
    await flushPromises()

    const tree = wrapper.findComponent({ name: 'UTree' })
    tree.vm.$emit('update:expanded', ['.', 'alpha'])
    await flushPromises()

    expect(browseMock).toHaveBeenLastCalledWith({ path: 'alpha' })
    expect(tree.props('items')[0].children[0].path).toBe('alpha/child')
  })

  it('emits the selected absolute path', async () => {
    const wrapper = await mountSuspended(BatchDirectoryBrowser, {
      props: {
        open: true
      }
    })
    await flushPromises()

    const tree = wrapper.findComponent({ name: 'UTree' })
    const alpha = tree.props('items')[0]
    tree.vm.$emit('update:modelValue', alpha)
    await nextTick()
    await flushPromises()

    expect(document.body.textContent).toContain('/batch-root/alpha')

    const useButton = wrapper.findAllComponents({ name: 'UButton' })
      .find(button => button.props('label') === 'Use directory')
    expect(useButton).toBeTruthy()
    await useButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('select')).toEqual([['/batch-root/alpha']])
    expect(wrapper.emitted('update:open')).toEqual([[false]])
  })
})
