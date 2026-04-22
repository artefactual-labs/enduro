import { expect, test } from 'vitest'

import {
  createDefaultBatchHints,
  createDefaultBatchStatus,
  resolveBatchPageData,
  toSelectOptions
} from './batch-page.helpers'

test('default batch helper values match the inactive page state', () => {
  expect(createDefaultBatchHints()).toEqual({ completedDirs: [] })
  expect(createDefaultBatchStatus()).toEqual({ running: false })
})

test('toSelectOptions maps processing configuration names to select items', () => {
  expect(toSelectOptions(['default', 'automated'])).toEqual([
    { label: 'default', value: 'default' },
    { label: 'automated', value: 'automated' }
  ])
})

test('resolveBatchPageData keeps fulfilled loader data', () => {
  const data = resolveBatchPageData({
    loadedPipelines: {
      status: 'fulfilled',
      value: [{ id: 'pipeline-1', name: 'am' }]
    },
    loadedHints: {
      status: 'fulfilled',
      value: { completedDirs: ['/completed'] }
    },
    loadedProcessingOptions: {
      status: 'fulfilled',
      value: ['default']
    },
    selectedPipelineId: 'pipeline-1'
  })

  expect(data.selectedPipelineId).toBe('pipeline-1')
  expect(data.pipelinesErrorMessage).toBe('')
  expect(data.hintsErrorMessage).toBe('')
  expect(data.processingErrorMessage).toBe('')
  expect(data.processingOptions).toEqual([{ label: 'default', value: 'default' }])
})

test('resolveBatchPageData falls back cleanly when API calls fail', () => {
  const data = resolveBatchPageData({
    loadedPipelines: {
      status: 'rejected',
      reason: new Error('no pipelines')
    },
    loadedHints: {
      status: 'rejected',
      reason: new Error('no hints')
    },
    loadedProcessingOptions: {
      status: 'rejected',
      reason: new Error('no configs')
    },
    selectedPipelineId: 'pipeline-2'
  })

  expect(data.pipelines).toEqual([])
  expect(data.hints).toEqual({ completedDirs: [] })
  expect(data.processingOptions).toEqual([])
  expect(data.pipelinesErrorMessage).toBe('Could not load configured pipelines.')
  expect(data.hintsErrorMessage).toBe('Could not load batch path hints.')
  expect(data.processingErrorMessage).toBe('Could not load processing configurations for the selected pipeline.')
})
