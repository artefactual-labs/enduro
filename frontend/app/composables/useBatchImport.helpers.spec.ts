import { expect, test } from 'vitest'

import {
  buildBatchSubmitRequest,
  parseSavedBatchDefaults
} from './useBatchImport.helpers'

test('parseSavedBatchDefaults normalizes saved local storage state', () => {
  const parsed = parseSavedBatchDefaults(JSON.stringify({
    completedDir: '/done',
    depth: 2,
    excludeHiddenFiles: true,
    processNameMetadata: true,
    rejectDuplicates: true,
    transferType: 'zipfile'
  }))

  expect(parsed).toEqual({
    completedDir: '/done',
    depth: 2,
    excludeHiddenFiles: true,
    mode: 'completed-dir',
    processNameMetadata: true,
    rejectDuplicates: true,
    retentionPeriod: '',
    transferType: 'zipfile'
  })
})

test('parseSavedBatchDefaults infers retention-period mode from legacy values and rejects invalid JSON', () => {
  const parsed = parseSavedBatchDefaults(JSON.stringify({
    completedDir: '',
    retentionPeriod: '72h'
  }))

  expect(parsed?.mode).toBe('retention-period')
  expect(parsed?.retentionPeriod).toBe('72h')
  expect(parseSavedBatchDefaults('{')).toBeNull()
})

test('buildBatchSubmitRequest composes the batch submit payload from UI state', () => {
  const request = buildBatchSubmitRequest({
    completedDir: '  /done  ',
    depth: -2,
    destinationMode: 'completed-dir',
    excludeHiddenFiles: true,
    path: ' /transfers ',
    pipelineName: 'am',
    processNameMetadata: true,
    processingConfig: 'automated',
    rejectDuplicates: false,
    retentionPeriod: '72h',
    transferType: 'zipfile'
  })

  expect(request).toEqual({
    completedDir: '/done',
    depth: 0,
    excludeHiddenFiles: true,
    path: '/transfers',
    pipeline: 'am',
    processNameMetadata: true,
    processingConfig: 'automated',
    rejectDuplicates: false,
    transferType: 'zipfile'
  })
})

test('buildBatchSubmitRequest uses retention period only in retention mode', () => {
  const request = buildBatchSubmitRequest({
    completedDir: '/done',
    depth: 0,
    destinationMode: 'retention-period',
    excludeHiddenFiles: false,
    path: '/transfers',
    pipelineName: null,
    processNameMetadata: false,
    processingConfig: '',
    rejectDuplicates: true,
    retentionPeriod: ' 72h ',
    transferType: ''
  })

  expect(request.completedDir).toBeUndefined()
  expect(request.retentionPeriod).toBe('72h')
})
