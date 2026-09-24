import { defineComponent, h } from 'vue'
import { DOMWrapper, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'

import AppSettings from './AppSettings.vue'
import AppFooter from './AppFooter.vue'

const Harness = defineComponent({
  setup() {
    return () => h('div', [h(AppSettings), h(AppFooter)])
  }
})

const body = new DOMWrapper(document.body)
let wrapper: Awaited<ReturnType<typeof mountSuspended>> | undefined

async function mountSettings() {
  wrapper = await mountSuspended(Harness, { attachTo: document.body })
  return wrapper
}

async function openSettings() {
  await wrapper!.get('[aria-label="Open settings"]').trigger('click')
  await vi.waitFor(() => expect(body.find('[role="dialog"]').exists()).toBe(true))
  await flushPromises()
  return body.get('[role="dialog"]')
}

function receiveStorageChange(key: string, value: string | null) {
  const oldValue = localStorage.getItem(key)
  if (value === null) localStorage.removeItem(key)
  else localStorage.setItem(key, value)
  window.dispatchEvent(new StorageEvent('storage', {
    key,
    oldValue,
    newValue: value,
    storageArea: localStorage
  }))
}

describe('AppSettings', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('nuxt-color-mode', 'light')
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = undefined
    document.body.innerHTML = ''
    localStorage.clear()
  })

  it('provides labelled controls and restores focus after Escape', async () => {
    await mountSettings()
    const trigger = wrapper!.get('[aria-label="Open settings"]')
    expect(trigger.attributes('aria-haspopup')).toBe('dialog')
    expect(trigger.attributes('aria-expanded')).toBe('false')

    const dialog = await openSettings()
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(document.getElementById(dialog.attributes('aria-labelledby') ?? '')?.textContent).toBe('Settings')
    expect(document.getElementById(dialog.attributes('aria-describedby') ?? '')?.textContent)
      .toBe('Preferences are saved in this browser.')

    const group = dialog.get('[role="radiogroup"]')
    expect(group.attributes('aria-label')).toBe('Appearance')
    expect(group.findAll('[role="radio"]')).toHaveLength(3)
    expect(group.text()).toContain('System')
    expect(group.text()).toContain('Light')
    expect(group.text()).toContain('Dark')
    expect(group.get('[value="light"]').attributes('aria-checked')).toBe('true')

    const control = dialog.get('[role="switch"]')
    expect(dialog.get(`label[for="${control.attributes('id')}"]`).text()).toBe('Show connection monitor')
    expect(document.getElementById(control.attributes('aria-describedby') ?? '')?.textContent)
      .toBe('Show connection status in the footer.')
    expect(control.attributes('aria-checked')).toBe('true')

    await dialog.trigger('keydown', { key: 'Escape' })
    await vi.waitFor(() => expect(body.find('[role="dialog"]').exists()).toBe(false))
    await vi.waitFor(() => expect(document.activeElement).toBe(trigger.element))
  })

  it('hides the monitor immediately and preserves both choices after remounting', async () => {
    await mountSettings()
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(true)
    const dialog = await openSettings()
    await dialog.get('[role="radio"][value="dark"]').trigger('click')
    await dialog.get('[role="switch"]').trigger('click')
    await flushPromises()

    expect(localStorage.getItem('nuxt-color-mode')).toBe('dark')
    expect(localStorage.getItem('enduro-show-connection-monitor')).toBe('false')
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(false)

    wrapper!.unmount()
    await mountSettings()
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(false)
    const restoredDialog = await openSettings()
    expect(restoredDialog.get('[role="radio"][value="dark"]').attributes('aria-checked')).toBe('true')
    expect(restoredDialog.get('[role="switch"]').attributes('aria-checked')).toBe('false')
    await restoredDialog.get('[role="switch"]').trigger('click')
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(true)
  })

  it('restores an existing theme preference', async () => {
    localStorage.setItem('nuxt-color-mode', 'dark')
    await mountSettings()
    const dialog = await openSettings()
    expect(dialog.get('[role="radio"][value="dark"]').attributes('aria-checked')).toBe('true')
  })

  it('lets users return to the system appearance and preserves that choice', async () => {
    await mountSettings()
    const dialog = await openSettings()
    for (const value of ['dark', 'light', 'system']) {
      const option = dialog.get(`[role="radio"][value="${value}"]`)
      await option.trigger('click')
      await flushPromises()
      expect(option.attributes('aria-checked')).toBe('true')
      expect(dialog.findAll('[role="radio"][aria-checked="true"]')).toHaveLength(1)
      expect(localStorage.getItem('nuxt-color-mode')).toBe(value)
    }

    wrapper!.unmount()
    await mountSettings()
    const restoredDialog = await openSettings()
    expect(restoredDialog.get('[role="radio"][value="system"]').attributes('aria-checked')).toBe('true')
  })

  it('supports arrow-key selection of the appearance', async () => {
    await mountSettings()
    const dialog = await openSettings()
    const light = dialog.get('[role="radio"][value="light"]')
    ;(light.element as HTMLElement).focus()
    await light.trigger('keydown', { key: 'ArrowRight' })
    await vi.waitFor(() => {
      expect(dialog.get('[role="radio"][value="dark"]').attributes('aria-checked')).toBe('true')
      expect(localStorage.getItem('nuxt-color-mode')).toBe('dark')
    })
  })

  it('updates the open controls and footer when another tab changes preferences', async () => {
    await mountSettings()
    const dialog = await openSettings()
    receiveStorageChange('nuxt-color-mode', 'dark')
    receiveStorageChange('enduro-show-connection-monitor', 'false')
    await flushPromises()

    expect(dialog.get('[role="radio"][value="dark"]').attributes('aria-checked')).toBe('true')
    expect(dialog.get('[role="switch"]').attributes('aria-checked')).toBe('false')
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(false)

    await dialog.get('[role="radio"][value="light"]').trigger('click')
    await flushPromises()
    expect(localStorage.getItem('nuxt-color-mode')).toBe('light')

    receiveStorageChange('enduro-show-connection-monitor', null)
    await flushPromises()
    expect(wrapper!.findComponent({ name: 'AppConnectionMonitor' }).exists()).toBe(true)
  })

  it('falls back to the system theme when a saved preference is invalid', async () => {
    localStorage.setItem('nuxt-color-mode', 'invalid')
    await mountSettings()
    const dialog = await openSettings()
    expect(localStorage.getItem('nuxt-color-mode')).toBe('system')
    expect(dialog.get('[role="radio"][value="system"]').attributes('aria-checked')).toBe('true')
  })
})
