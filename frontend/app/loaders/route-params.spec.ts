import { expect, test } from 'vitest'

import { normalizeRouteStringParam, parseCollectionId } from './route-params'

test('normalizeRouteStringParam reads string values from route params and query arrays', () => {
  expect(normalizeRouteStringParam('abc')).toBe('abc')
  expect(normalizeRouteStringParam(['def', 'ghi'])).toBe('def')
  expect(normalizeRouteStringParam(undefined)).toBe('')
  expect(normalizeRouteStringParam(42)).toBe('')
})

test('parseCollectionId accepts safe integer strings and rejects invalid values', () => {
  expect(parseCollectionId('17')).toBe(17)
  expect(parseCollectionId(['23'])).toBe(23)
  expect(parseCollectionId('abc')).toBe(0)
  expect(parseCollectionId('-1')).toBe(0)
  expect(parseCollectionId(String(Number.MAX_SAFE_INTEGER + 1))).toBe(0)
})
