import { expect, test } from 'vitest'
import {
  BulkRequestBodyOperationEnum,
  BulkRequestBodyStatusEnum
} from '../openapi-generator'

import {
  buildBulkRequest,
  createDefaultBulkStatus,
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
