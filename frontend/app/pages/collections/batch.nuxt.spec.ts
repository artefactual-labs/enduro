import { nextTick, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mountSuspended, mockNuxtImport } from '@nuxt/test-utils/runtime'

import BatchPage from './batch.vue'

const { useBatchImportMock } = vi.hoisted(() => ({
  useBatchImportMock: vi.fn()
}))

mockNuxtImport('useBatchImport', () => useBatchImportMock)

function createBatchImportState(overrides: Partial<ReturnType<typeof useBatchImportMock>> = {}) {
  return {
    canSubmit: ref(true),
    completedDir: ref(''),
    destinationMode: ref('completed-dir'),
    destinationModeOptions: [{ label: 'Completed directory', value: 'completed-dir' }],
    depth: ref(0),
    excludeHiddenFiles: ref(false),
    hasKnownCompletedDirs: ref(false),
    hints: ref({ completedDirs: [] }),
    hintsErrorMessage: ref(''),
    isLoadingHints: ref(false),
    isLoadingPipelines: ref(false),
    isLoadingProcessing: ref(false),
    isLoadingStatus: ref(false),
    isRunning: ref(false),
    isSubmitting: ref(false),
    path: ref('/Users/jesus/Projects/enduro-legacy/tmp.ignored/transfers/batch-with-folders'),
    pipelineOptions: ref([{ label: 'Archivematica', value: 'am' }]),
    pipelinesErrorMessage: ref(''),
    processNameMetadata: ref(false),
    processingErrorMessage: ref(''),
    processingOptions: ref([{ label: 'Local storage', value: 'local-storage' }]),
    rejectDuplicates: ref(false),
    retentionPeriod: ref(''),
    selectedPipelineId: ref('am'),
    selectedProcessingConfig: ref('local-storage'),
    selectedTransferType: ref(''),
    showBatchStatus: ref(false),
    status: ref({ running: false }),
    statusErrorMessage: ref(''),
    loadStatus: vi.fn(),
    submit: vi.fn(),
    submitErrorMessage: ref(''),
    transferOptions: [{ label: 'Standard', value: 'standard' }],
    useCompletedDirHint: vi.fn(),
    ...overrides
  }
}

describe('batch import page', () => {
  beforeEach(() => {
    useBatchImportMock.mockReset()
  })

  it('confirms the path and processing configuration before submitting', async () => {
    const submit = vi.fn()
    useBatchImportMock.mockReturnValue(createBatchImportState({ submit }))

    const wrapper = await mountSuspended(BatchPage, {
      route: '/collections/batch'
    })

    const submitButton = wrapper.findAll('button').find(node => node.text() === 'Submit')
    expect(submitButton).toBeTruthy()
    await submitButton?.trigger('click')
    await nextTick()

    expect(document.body.textContent).toContain('Submit batch import?')
    expect(document.body.textContent).toContain('Review the batch details before submitting.')
    expect(document.body.textContent).toContain('You\'re submitting this directory as a batch transfer.')
    expect(document.body.textContent).toContain('Path')
    expect(document.body.textContent).toContain('/Users/jesus/Projects/enduro-legacy/tmp.ignored/transfers/batch-with-folders')
    expect(document.body.textContent).toContain('Processing configuration')
    expect(document.body.textContent).toContain('Local storage')
    expect(submit).not.toHaveBeenCalled()

    const dialog = wrapper.findComponent({ name: 'AppConfirmDialog' })
    expect(dialog.props('open')).toBe(true)
    expect(dialog.props('modalClass')).toBe('max-w-lg')
    expect(dialog.props('description')).toBe('Review the batch details before submitting.')
    expect(document.body.querySelector('pre')?.textContent).toBe('/Users/jesus/Projects/enduro-legacy/tmp.ignored/transfers/batch-with-folders')
    dialog.vm.$emit('confirm')
    await nextTick()

    expect(submit).toHaveBeenCalledOnce()
  })
})
