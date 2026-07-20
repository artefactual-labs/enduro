import { describe, expect, it } from 'vitest'
import {
  type EnduroCollectionStatusTransition,
  EnduroCollectionStatusTransitionStatusEnum as Status
} from '../openapi-generator'

import {
  buildCollectionStatusPeriods,
  formatCollectionStatusDuration,
  summarizeCollectionStatusPeriods
} from './collection-status-history'

function transition(id: number, status: Status, occurredAt: string, isRunStart = false): EnduroCollectionStatusTransition {
  return {
    id,
    status,
    occurredAt: new Date(occurredAt),
    isRunStart,
    runId: 'run-42',
    workflowId: 'workflow-42'
  }
}

describe('collection status history', () => {
  it('builds repeated status periods and a live final period', () => {
    const periods = buildCollectionStatusPeriods([
      transition(1, Status.Queued, '2026-07-19T10:00:00Z', true),
      transition(2, Status.InProgress, '2026-07-19T10:10:00Z'),
      transition(3, Status.Pending, '2026-07-19T10:20:00Z'),
      transition(4, Status.InProgress, '2026-07-19T10:50:00Z'),
      transition(5, Status.Error, '2026-07-19T11:00:00Z')
    ], 'run-42', new Date('2026-07-19T12:00:00Z'))

    expect(periods.map(period => period.status)).toEqual([
      'queued',
      'in progress',
      'pending',
      'in progress',
      'error'
    ])
    expect(periods[0]?.durationMs).toBe(10 * 60 * 1000)
    expect(periods[4]).toMatchObject({ isCurrent: true, durationMs: 60 * 60 * 1000 })

    expect(summarizeCollectionStatusPeriods(periods)).toEqual([
      { status: 'queued', durationMs: 10 * 60 * 1000 },
      { status: 'in progress', durationMs: 20 * 60 * 1000 },
      { status: 'pending', durationMs: 30 * 60 * 1000 },
      { status: 'error', durationMs: 60 * 60 * 1000 }
    ])
  })

  it('filters other runs and invalid dates', () => {
    const otherRun = { ...transition(2, Status.Error, '2026-07-19T11:00:00Z'), runId: 'other-run' }
    const invalid = { ...transition(3, Status.Pending, 'invalid'), occurredAt: new Date('invalid') }

    const periods = buildCollectionStatusPeriods([
      transition(1, Status.Queued, '2026-07-19T10:00:00Z', true),
      otherRun,
      invalid
    ], 'run-42', new Date('2026-07-19T10:05:00Z'))

    expect(periods).toHaveLength(1)
    expect(periods[0]?.durationMs).toBe(5 * 60 * 1000)
  })

  it('formats durations compactly', () => {
    expect(formatCollectionStatusDuration(250)).toBe('<1s')
    expect(formatCollectionStatusDuration(45_000)).toBe('45s')
    expect(formatCollectionStatusDuration(75 * 60 * 1000)).toBe('1h 15m')
    expect(formatCollectionStatusDuration((25 * 60 + 5) * 60 * 1000)).toBe('1d 1h 5m')
  })

  it('does not count an open completed state as workflow time', () => {
    const periods = buildCollectionStatusPeriods([
      transition(1, Status.InProgress, '2026-07-19T10:00:00Z', true),
      transition(2, Status.Done, '2026-07-19T10:30:00Z')
    ], 'run-42', new Date('2026-07-20T10:30:00Z'))

    expect(summarizeCollectionStatusPeriods(periods)).toEqual([
      { status: 'in progress', durationMs: 30 * 60 * 1000 }
    ])
  })
})
