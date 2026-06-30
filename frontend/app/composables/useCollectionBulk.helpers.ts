import type { BulkRequestBody, BulkStatusResult } from '../openapi-generator'
import {
  BulkRequestBodyOperationEnum,
  BulkRequestBodyStatusEnum
} from '../openapi-generator'

export type BulkSelectOption = {
  disabled?: boolean
  label: string
  value: string
}

export function createDefaultBulkStatus(): BulkStatusResult {
  return {
    running: false
  }
}

export function didBulkRunFail(status: string | null | undefined): boolean {
  return status ? status !== 'completed' : false
}

export function buildBulkRequest(input: {
  operation: BulkRequestBody['operation']
  size: number | null
  status: BulkRequestBody['status']
}): BulkRequestBody {
  const request: BulkRequestBody = {
    operation: input.operation,
    status: input.status
  }

  if (typeof input.size === 'number' && Number.isFinite(input.size) && input.size > 0) {
    request.size = Math.trunc(input.size)
  }

  return request
}

export function buildBulkOperationOptions(status: BulkRequestBody['status']): BulkSelectOption[] {
  return [
    {
      label: 'Retry',
      value: BulkRequestBodyOperationEnum.Retry,
      disabled: status === BulkRequestBodyStatusEnum.Queued
    },
    {
      label: 'Cancel',
      value: BulkRequestBodyOperationEnum.Cancel,
      disabled: status !== BulkRequestBodyStatusEnum.Queued
    },
    {
      label: 'Abandon',
      value: BulkRequestBodyOperationEnum.Abandon,
      disabled: status !== BulkRequestBodyStatusEnum.Pending
    }
  ]
}

export function defaultBulkOperationForStatus(status: BulkRequestBody['status']): BulkRequestBody['operation'] {
  return status === BulkRequestBodyStatusEnum.Queued
    ? BulkRequestBodyOperationEnum.Cancel
    : BulkRequestBodyOperationEnum.Retry
}
