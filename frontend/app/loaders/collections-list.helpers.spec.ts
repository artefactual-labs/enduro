import { expect, test } from 'vitest'

import {
  buildCollectionListRequest,
  normalizeDateFilter,
  normalizeFieldFilter,
  normalizeStatusFilter,
  readCollectionsQueryValue,
  resolveEarliestCreatedTime
} from './collections-list.helpers'

test('readCollectionsQueryValue normalizes route query strings and arrays', () => {
  expect(readCollectionsQueryValue('done')).toBe('done')
  expect(readCollectionsQueryValue(['pending', 'error'])).toBe('pending')
  expect(readCollectionsQueryValue(null)).toBe('')
})

test('collections list filters normalize invalid values to defaults', () => {
  expect(normalizeStatusFilter('done')).toBe('done')
  expect(normalizeStatusFilter('bogus')).toBe('all')
  expect(normalizeDateFilter('24h')).toBe('24h')
  expect(normalizeDateFilter('bogus')).toBe('all')
  expect(normalizeFieldFilter('aip_id')).toBe('aip_id')
  expect(normalizeFieldFilter('bogus')).toBe('name')
})

test('resolveEarliestCreatedTime subtracts hours and days from a reference date', () => {
  const now = new Date('2026-04-23T10:00:00.000Z')

  expect(resolveEarliestCreatedTime('all', now)).toBeUndefined()
  expect(resolveEarliestCreatedTime('6h', now)?.toISOString()).toBe('2026-04-23T04:00:00.000Z')
  expect(resolveEarliestCreatedTime('3d', now)?.toISOString()).toBe('2026-04-20T10:00:00.000Z')
})

test('buildCollectionListRequest maps filters and search query into API request fields', () => {
  const result = buildCollectionListRequest({
    status: 'in progress',
    date: '24h',
    field: 'transfer_id',
    q: '31ceb5d5-a9c1-488b-b4ee-40910e54109e',
    cursor: 'next-page'
  }, new Date('2026-04-23T10:00:00.000Z'))

  expect(result.invalidQuery).toBe(false)
  expect(result.request.cursor).toBe('next-page')
  expect(result.request.transferId).toBe('31ceb5d5-a9c1-488b-b4ee-40910e54109e')
  expect(result.request.status).toBe('in progress')
  expect(result.request.earliestCreatedTime?.toISOString()).toBe('2026-04-22T10:00:00.000Z')
})

test('buildCollectionListRequest rejects invalid UUID searches for UUID-only fields', () => {
  const result = buildCollectionListRequest({
    field: 'pipeline_id',
    q: 'not-a-uuid'
  })

  expect(result.invalidQuery).toBe(true)
  expect(result.request).toEqual({})
})

test('buildCollectionListRequest leaves original ID searches as exact free text', () => {
  const result = buildCollectionListRequest({
    field: 'original_id',
    q: 'DPJ-SIP-97'
  })

  expect(result.invalidQuery).toBe(false)
  expect(result.request.originalId).toBe('DPJ-SIP-97')
})
