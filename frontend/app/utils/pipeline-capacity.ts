export type PipelineCapacitySummary = {
  available: number
  badgeLabel: string
  capacity: number
  color: 'success' | 'warning' | 'error' | 'neutral'
  current: number
  currentClamped: number
  currentLabel: string
  hasCapacity: boolean
}

function normalizeCount(value: number | null | undefined): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return 0
  return Math.max(0, Math.floor(value))
}

export function summarizePipelineCapacity(
  currentValue: number | null | undefined,
  capacityValue: number | null | undefined
): PipelineCapacitySummary {
  const current = normalizeCount(currentValue)
  const capacity = normalizeCount(capacityValue)

  if (capacity <= 0) {
    return {
      available: 0,
      badgeLabel: 'Unavailable',
      capacity: 0,
      color: 'neutral',
      current,
      currentClamped: 0,
      currentLabel: 'No slots configured',
      hasCapacity: false
    }
  }

  const currentClamped = Math.min(current, capacity)
  const available = Math.max(capacity - currentClamped, 0)
  let color: PipelineCapacitySummary['color'] = 'success'
  if (currentClamped >= capacity) {
    color = 'error'
  } else if (currentClamped * 3 >= capacity * 2) {
    color = 'warning'
  }

  return {
    available,
    badgeLabel: available === 1 ? '1 available' : `${available} available`,
    capacity,
    color,
    current,
    currentClamped,
    currentLabel: currentClamped === 1 ? '1 slot in use' : `${currentClamped} slots in use`,
    hasCapacity: true
  }
}
