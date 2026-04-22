import type { BulkRequestBody, BulkStatusResult } from '../openapi-generator'

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
