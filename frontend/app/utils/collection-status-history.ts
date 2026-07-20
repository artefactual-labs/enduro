import type { EnduroCollectionStatusTransition } from '~/openapi-generator'

export type CollectionStatusPeriod = {
  id: number
  status: string
  previousStatus: string | null
  reason: string | null
  workflowId: string
  runId: string
  startedAt: Date
  endedAt: Date | null
  durationMs: number
  isCurrent: boolean
  isRunStart: boolean
}

export type CollectionStatusTotal = {
  status: string
  durationMs: number
}

const openTerminalStatuses = new Set(['done', 'abandoned'])

export function collectionStatusPeriodHasDuration(period: CollectionStatusPeriod): boolean {
  return !(period.isCurrent && openTerminalStatuses.has(period.status))
}

function validTimestamp(value: Date): number | null {
  const timestamp = value.getTime()
  return Number.isFinite(timestamp) ? timestamp : null
}

export function buildCollectionStatusPeriods(
  transitions: EnduroCollectionStatusTransition[],
  runId: string | null | undefined,
  now = new Date()
): CollectionStatusPeriod[] {
  if (!runId) return []

  const ordered = transitions
    .filter(transition => transition.runId === runId && validTimestamp(transition.occurredAt) !== null)
    .toSorted((left, right) => {
      const timeDifference = left.occurredAt.getTime() - right.occurredAt.getTime()
      return timeDifference || left.id - right.id
    })

  const nowTimestamp = validTimestamp(now) ?? Date.now()

  return ordered.map((transition, index) => {
    const next = ordered[index + 1]
    const startedAt = transition.occurredAt
    const endedAt = next?.occurredAt ?? null
    const isCurrent = !next
    const endTimestamp = endedAt?.getTime() ?? nowTimestamp

    return {
      id: transition.id,
      status: transition.status,
      previousStatus: transition.previousStatus ?? null,
      reason: transition.reason ?? null,
      workflowId: transition.workflowId,
      runId: transition.runId,
      startedAt,
      endedAt,
      durationMs: Math.max(0, endTimestamp - startedAt.getTime()),
      isCurrent,
      isRunStart: transition.isRunStart
    }
  })
}

export function summarizeCollectionStatusPeriods(periods: CollectionStatusPeriod[]): CollectionStatusTotal[] {
  const totals = new Map<string, number>()
  for (const period of periods) {
    if (!collectionStatusPeriodHasDuration(period)) continue
    totals.set(period.status, (totals.get(period.status) ?? 0) + period.durationMs)
  }

  const statusOrder = ['queued', 'in progress', 'pending', 'error', 'done', 'abandoned']
  return [...totals.entries()]
    .map(([status, durationMs]) => ({ status, durationMs }))
    .toSorted((left, right) => {
      const leftIndex = statusOrder.indexOf(left.status)
      const rightIndex = statusOrder.indexOf(right.status)
      return (leftIndex === -1 ? statusOrder.length : leftIndex) - (rightIndex === -1 ? statusOrder.length : rightIndex)
    })
}

export function formatCollectionStatusDuration(durationMs: number): string {
  if (durationMs > 0 && durationMs < 1000) return '<1s'

  const totalSeconds = Math.max(0, Math.floor(durationMs / 1000))
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  const segments: string[] = []
  if (days) segments.push(`${days}d`)
  if (hours) segments.push(`${hours}h`)
  if (minutes) segments.push(`${minutes}m`)
  if (!days && !hours && !minutes) segments.push(`${seconds}s`)

  return segments.join(' ')
}
