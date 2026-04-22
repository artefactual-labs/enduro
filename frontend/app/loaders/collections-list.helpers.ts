import {
  CollectionListStatusEnum,
  type CollectionApiCollectionListRequest
} from '../openapi-generator'

export type StatusFilter = 'all' | 'error' | 'done' | 'in progress' | 'queued' | 'pending' | 'abandoned'
export type DateFilter = 'all' | '3h' | '6h' | '24h' | '3d' | '14d' | '30d'
export type FieldFilter = 'name' | 'pipeline_id' | 'transfer_id' | 'aip_id' | 'original_id'
export type CollectionFilterOption<T extends string> = {
  label: string
  value: T
}

export const statusOptions: Array<CollectionFilterOption<StatusFilter>> = [
  { label: 'Status', value: 'all' },
  { label: 'Error', value: 'error' },
  { label: 'Done', value: 'done' },
  { label: 'In progress', value: 'in progress' },
  { label: 'Queued', value: 'queued' },
  { label: 'Pending', value: 'pending' },
  { label: 'Abandoned', value: 'abandoned' }
]

export const dateOptions: Array<CollectionFilterOption<DateFilter>> = [
  { label: 'Creation date', value: 'all' },
  { label: 'Last 3 hours', value: '3h' },
  { label: 'Last 6 hours', value: '6h' },
  { label: 'Last 24 hours', value: '24h' },
  { label: 'Last 3 days', value: '3d' },
  { label: 'Last 14 days', value: '14d' },
  { label: 'Last 30 days', value: '30d' }
]

export const fieldOptions: Array<CollectionFilterOption<FieldFilter>> = [
  { label: 'Name', value: 'name' },
  { label: 'Pipeline ID', value: 'pipeline_id' },
  { label: 'Transfer ID', value: 'transfer_id' },
  { label: 'AIP ID', value: 'aip_id' },
  { label: 'Original ID', value: 'original_id' }
]

const uuidFieldSet = new Set<FieldFilter>(['pipeline_id', 'transfer_id', 'aip_id'])
const statusOptionValues = new Set(statusOptions.map(option => option.value))
const dateOptionValues = new Set(dateOptions.map(option => option.value))
const fieldOptionValues = new Set(fieldOptions.map(option => option.value))
const uuidLikeRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const statusToApiStatus = {
  'error': CollectionListStatusEnum.Error,
  'done': CollectionListStatusEnum.Done,
  'in progress': CollectionListStatusEnum.InProgress,
  'queued': CollectionListStatusEnum.Queued,
  'pending': CollectionListStatusEnum.Pending,
  'abandoned': CollectionListStatusEnum.Abandoned
} as const

export function readCollectionsQueryValue(value: unknown): string {
  if (typeof value === 'string') return value
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0]
  return ''
}

export function normalizeStatusFilter(value: string): StatusFilter {
  return statusOptionValues.has(value as StatusFilter) ? value as StatusFilter : 'all'
}

export function normalizeDateFilter(value: string): DateFilter {
  return dateOptionValues.has(value as DateFilter) ? value as DateFilter : 'all'
}

export function normalizeFieldFilter(value: string): FieldFilter {
  return fieldOptionValues.has(value as FieldFilter) ? value as FieldFilter : 'name'
}

export function isValidCollectionSearchQuery(field: FieldFilter, value: string): boolean {
  if (!value) return true
  if (!uuidFieldSet.has(field)) return true
  return uuidLikeRegex.test(value)
}

export function resolveEarliestCreatedTime(filter: DateFilter, now: Date = new Date()): Date | undefined {
  if (filter === 'all') return undefined

  const match = /^(\d+)([dh])$/.exec(filter)
  if (!match) return undefined

  const amount = Number.parseInt(match[1] ?? '0', 10)
  const unit = match[2]
  if (!Number.isFinite(amount) || amount <= 0) return undefined

  const ms = unit === 'h' ? amount * 60 * 60 * 1000 : amount * 24 * 60 * 60 * 1000
  return new Date(now.getTime() - ms)
}

export function buildCollectionListRequest(
  queryLike: Record<string, unknown>,
  now: Date = new Date()
): {
  invalidQuery: boolean
  request: CollectionApiCollectionListRequest
} {
  const status = normalizeStatusFilter(readCollectionsQueryValue(queryLike.status))
  const date = normalizeDateFilter(readCollectionsQueryValue(queryLike.date))
  const field = normalizeFieldFilter(readCollectionsQueryValue(queryLike.field))
  const query = readCollectionsQueryValue(queryLike.q).trim()
  const cursor = readCollectionsQueryValue(queryLike.cursor)

  if (!isValidCollectionSearchQuery(field, query)) {
    return {
      invalidQuery: true,
      request: {}
    }
  }

  const request: CollectionApiCollectionListRequest = {}
  if (cursor) request.cursor = cursor

  const earliestCreatedTime = resolveEarliestCreatedTime(date, now)
  if (earliestCreatedTime) request.earliestCreatedTime = earliestCreatedTime

  if (status !== 'all') request.status = statusToApiStatus[status]

  if (query) {
    switch (field) {
      case 'name':
        request.name = query
        break
      case 'pipeline_id':
        request.pipelineId = query
        break
      case 'transfer_id':
        request.transferId = query
        break
      case 'aip_id':
        request.aipId = query
        break
      case 'original_id':
        request.originalId = query
        break
    }
  }

  return {
    invalidQuery: false,
    request
  }
}
