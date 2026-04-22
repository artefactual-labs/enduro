import { expect, test } from 'vitest'

import {
  defaultDashboardUiOptions,
  normalizeDashboardUiOptions,
  parseSavedDashboardUiOptions
} from './useDashboardUiOptions.helpers'

test('normalizeDashboardUiOptions fills missing UI options from defaults', () => {
  expect(normalizeDashboardUiOptions({})).toEqual(defaultDashboardUiOptions)
  expect(normalizeDashboardUiOptions({
    collectionsSearchOpen: true
  })).toEqual({
    collectionsSearchOpen: true
  })
})

test('parseSavedDashboardUiOptions reads saved local UI options and rejects invalid JSON', () => {
  expect(parseSavedDashboardUiOptions(JSON.stringify({
    collectionsSearchOpen: true
  }))).toEqual({
    collectionsSearchOpen: true
  })

  expect(parseSavedDashboardUiOptions('{')).toBeNull()
  expect(parseSavedDashboardUiOptions(null)).toBeNull()
})
