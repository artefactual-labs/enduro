import { describe, expect, it, vi } from 'vitest'
import {
  EnduroCollectionStatusHistoryAvailabilityEnum as Availability
} from '../openapi-generator'

import { loadCollectionWorkflowSources } from './collection-workflow.helpers'

describe('collection workflow sources', () => {
  it('keeps status history when Temporal fails', async () => {
    const statusHistory = { availability: Availability.Unavailable, transitions: [] }
    const collections = {
      workflow: vi.fn().mockRejectedValue(new Error('Temporal unavailable')),
      statusHistory: vi.fn().mockResolvedValue(statusHistory)
    }

    const got = await loadCollectionWorkflowSources(collections, 42)

    expect(got).toEqual({
      workflow: null,
      workflowError: 'The Temporal workflow history is not available.',
      statusHistory,
      statusHistoryError: ''
    })
  })

  it('keeps Temporal data when status history fails', async () => {
    const workflow = { status: 'ACTIVE', history: [] }
    const collections = {
      workflow: vi.fn().mockResolvedValue(workflow),
      statusHistory: vi.fn().mockRejectedValue(new Error('Database unavailable'))
    }

    const got = await loadCollectionWorkflowSources(collections, 42)

    expect(got).toEqual({
      workflow,
      workflowError: '',
      statusHistory: null,
      statusHistoryError: 'The collection status history could not be loaded.'
    })
  })
})
