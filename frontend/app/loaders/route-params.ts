export function parseCollectionId(value: unknown): number {
  const normalized = normalizeRouteStringParam(value)
  if (!/^\d+$/.test(normalized)) return 0

  const parsed = Number(normalized)
  return Number.isSafeInteger(parsed) ? parsed : 0
}

export function normalizeRouteStringParam(value: unknown): string {
  if (typeof value === 'string') return value
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0]
  return ''
}
