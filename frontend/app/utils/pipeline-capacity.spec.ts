import { expect, test } from 'vitest'

import { summarizePipelineCapacity } from './pipeline-capacity'

test('summarizePipelineCapacity reports available slots with success color when mostly empty', () => {
  const summary = summarizePipelineCapacity(0, 3)

  expect(summary.currentClamped).toBe(0)
  expect(summary.available).toBe(3)
  expect(summary.badgeLabel).toBe('3 available')
  expect(summary.color).toBe('success')
})

test('summarizePipelineCapacity reports warning when nearing capacity', () => {
  const summary = summarizePipelineCapacity(2, 3)

  expect(summary.currentClamped).toBe(2)
  expect(summary.available).toBe(1)
  expect(summary.color).toBe('warning')
})

test('summarizePipelineCapacity reports error when full', () => {
  const summary = summarizePipelineCapacity(3, 3)

  expect(summary.currentClamped).toBe(3)
  expect(summary.available).toBe(0)
  expect(summary.color).toBe('error')
})

test('summarizePipelineCapacity reports neutral when capacity is unavailable', () => {
  const summary = summarizePipelineCapacity(2, 0)

  expect(summary.hasCapacity).toBe(false)
  expect(summary.badgeLabel).toBe('Unavailable')
  expect(summary.currentLabel).toBe('No slots configured')
  expect(summary.color).toBe('neutral')
})

test('summarizePipelineCapacity clamps negative and overflowing values', () => {
  const summary = summarizePipelineCapacity(-2, 1.9)

  expect(summary.current).toBe(0)
  expect(summary.capacity).toBe(1)
  expect(summary.currentClamped).toBe(0)
  expect(summary.currentLabel).toBe('0 slots in use')
})
