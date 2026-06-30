import { expect, test } from 'vitest'
import {
  BulkRequestBodyOperationEnum,
  BulkRequestBodyStatusEnum
} from '../openapi-generator'

import {
  buildBulkOperationOptions,
  buildBulkRequest,
  createDefaultBulkStatus,
  defaultBulkOperationForStatus,
  didBulkRunFail
} from './useCollectionBulk.helpers'

test('createDefaultBulkStatus returns an idle bulk status', () => {
  expect(createDefaultBulkStatus()).toEqual({ running: false })
})

test('didBulkRunFail only treats non-completed statuses as failures', () => {
  expect(didBulkRunFail('completed')).toBe(false)
  expect(didBulkRunFail('failed')).toBe(true)
  expect(didBulkRunFail(undefined)).toBe(false)
})

test('buildBulkRequest includes positive size limits and omits invalid ones', () => {
  expect(buildBulkRequest({
    operation: BulkRequestBodyOperationEnum.Retry,
    size: 12.8,
    status: BulkRequestBodyStatusEnum.Error
  })).toEqual({
    operation: BulkRequestBodyOperationEnum.Retry,
    size: 12,
    status: BulkRequestBodyStatusEnum.Error
  })

  expect(buildBulkRequest({
    operation: BulkRequestBodyOperationEnum.Retry,
    size: 0,
    status: BulkRequestBodyStatusEnum.Error
  })).toEqual({
    operation: BulkRequestBodyOperationEnum.Retry,
    status: BulkRequestBodyStatusEnum.Error
  })
})

test('buildBulkOperationOptions enables operations for compatible statuses only', () => {
  expect(buildBulkOperationOptions(BulkRequestBodyStatusEnum.Pending)).toEqual([
    { label: 'Retry', value: BulkRequestBodyOperationEnum.Retry, disabled: false },
    { label: 'Cancel', value: BulkRequestBodyOperationEnum.Cancel, disabled: true },
    { label: 'Abandon', value: BulkRequestBodyOperationEnum.Abandon, disabled: false }
  ])

  expect(buildBulkOperationOptions(BulkRequestBodyStatusEnum.Error)).toEqual([
    { label: 'Retry', value: BulkRequestBodyOperationEnum.Retry, disabled: false },
    { label: 'Cancel', value: BulkRequestBodyOperationEnum.Cancel, disabled: true },
    { label: 'Abandon', value: BulkRequestBodyOperationEnum.Abandon, disabled: true }
  ])

  expect(buildBulkOperationOptions(BulkRequestBodyStatusEnum.Queued)).toEqual([
    { label: 'Retry', value: BulkRequestBodyOperationEnum.Retry, disabled: true },
    { label: 'Cancel', value: BulkRequestBodyOperationEnum.Cancel, disabled: false },
    { label: 'Abandon', value: BulkRequestBodyOperationEnum.Abandon, disabled: true }
  ])
})

test('defaultBulkOperationForStatus selects cancel for queued collections', () => {
  expect(defaultBulkOperationForStatus(BulkRequestBodyStatusEnum.Queued)).toBe(BulkRequestBodyOperationEnum.Cancel)
  expect(defaultBulkOperationForStatus(BulkRequestBodyStatusEnum.Pending)).toBe(BulkRequestBodyOperationEnum.Retry)
})
