import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import BulkPage from './bulk.vue'

const { useCollectionBulkMock } = vi.hoisted(() => ({
  useCollectionBulkMock: vi.fn()
}))

mockNuxtImport('useCollectionBulk', () => useCollectionBulkMock)

function createCollectionBulkState() {
  return {
    canSubmit: ref(true),
    hasCompletedRun: ref(false),
    isLoadingStatus: ref(false),
    isRunning: ref(false),
    isSubmitting: ref(false),
    lastRunFailed: ref(false),
    loadStatus: vi.fn(),
    operationOptions: [
      { label: 'Retry', value: 'retry' },
      { label: 'Cancel', value: 'cancel', disabled: true },
      { label: 'Abandon', value: 'abandon', disabled: true }
    ],
    selectedOperation: ref('retry'),
    selectedStatus: ref('error'),
    size: ref<number | null>(null),
    status: ref({ running: false }),
    statusErrorMessage: ref(''),
    statusOptions: [{ label: 'Error', value: 'error' }],
    submit: vi.fn(),
    submitErrorMessage: ref(''),
    submitSuccessMessage: ref('')
  }
}

describe('bulk operation page', () => {
  beforeEach(() => {
    useCollectionBulkMock.mockReset()
  })

  it('shows legacy bulk operation choices including disabled cancel and abandon options', async () => {
    useCollectionBulkMock.mockReturnValue(createCollectionBulkState())

    const wrapper = await mountSuspended(BulkPage, {
      route: '/collections/bulk'
    })

    const selects = wrapper.findAllComponents({ name: 'USelect' })
    const operationSelect = selects[1]

    expect(operationSelect).toBeTruthy()
    expect(operationSelect?.props('items')).toEqual([
      { label: 'Retry', value: 'retry' },
      { label: 'Cancel', value: 'cancel', disabled: true },
      { label: 'Abandon', value: 'abandon', disabled: true }
    ])
  })
})
