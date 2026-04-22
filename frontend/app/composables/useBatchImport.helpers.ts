import type { SubmitRequestBody } from '../openapi-generator'

export type DefaultsMode = 'completed-dir' | 'retention-period'

export type SavedBatchDefaults = {
  completedDir: string
  depth: number
  excludeHiddenFiles: boolean
  mode: DefaultsMode
  processNameMetadata: boolean
  rejectDuplicates: boolean
  retentionPeriod: string
  transferType: string
}

export function parseSavedBatchDefaults(rawDefaults: string | null): SavedBatchDefaults | null {
  if (!rawDefaults) return null

  try {
    const parsed = JSON.parse(rawDefaults) as Partial<SavedBatchDefaults>
    const completedDir = typeof parsed.completedDir === 'string' ? parsed.completedDir : ''
    const retentionPeriod = typeof parsed.retentionPeriod === 'string' ? parsed.retentionPeriod : ''

    return {
      completedDir,
      depth: typeof parsed.depth === 'number' && Number.isFinite(parsed.depth) ? parsed.depth : 0,
      excludeHiddenFiles: Boolean(parsed.excludeHiddenFiles),
      mode: parsed.mode === 'retention-period'
        || (!parsed.mode && retentionPeriod.length > 0 && completedDir.length === 0)
        ? 'retention-period'
        : 'completed-dir',
      processNameMetadata: Boolean(parsed.processNameMetadata),
      rejectDuplicates: Boolean(parsed.rejectDuplicates),
      retentionPeriod,
      transferType: typeof parsed.transferType === 'string' ? parsed.transferType : ''
    }
  } catch {
    return null
  }
}

export function buildBatchSubmitRequest(input: {
  completedDir: string
  depth: number
  destinationMode: DefaultsMode
  excludeHiddenFiles: boolean
  path: string
  pipelineName: string | null
  processNameMetadata: boolean
  processingConfig: string
  rejectDuplicates: boolean
  retentionPeriod: string
  transferType: string
}): SubmitRequestBody {
  const request: SubmitRequestBody = {
    depth: Number.isFinite(input.depth) ? Math.max(0, input.depth) : 0,
    excludeHiddenFiles: input.excludeHiddenFiles,
    path: input.path.trim(),
    processNameMetadata: input.processNameMetadata,
    rejectDuplicates: input.rejectDuplicates
  }

  if (input.pipelineName) {
    request.pipeline = input.pipelineName
  }

  if (input.processingConfig) {
    request.processingConfig = input.processingConfig
  }

  if (input.transferType) {
    request.transferType = input.transferType
  }

  const trimmedCompletedDir = input.completedDir.trim()
  const trimmedRetentionPeriod = input.retentionPeriod.trim()

  if (input.destinationMode === 'completed-dir' && trimmedCompletedDir) {
    request.completedDir = trimmedCompletedDir
  }

  if (input.destinationMode === 'retention-period' && trimmedRetentionPeriod) {
    request.retentionPeriod = trimmedRetentionPeriod
  }

  return request
}
