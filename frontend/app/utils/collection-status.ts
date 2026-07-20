export type CollectionStatusColor = 'neutral' | 'info' | 'warning' | 'success' | 'error'

export type CollectionStatusPresentation = {
  color: CollectionStatusColor
  icon: string
  label: string
}

const statusPresentations: Record<string, CollectionStatusPresentation> = {
  'queued': { color: 'info', icon: 'i-lucide-clock-3', label: 'Queued' },
  'in progress': { color: 'warning', icon: 'i-lucide-loader-circle', label: 'In progress' },
  'pending': { color: 'info', icon: 'i-lucide-circle-help', label: 'Pending' },
  'done': { color: 'success', icon: 'i-lucide-circle-check', label: 'Done' },
  'error': { color: 'error', icon: 'i-lucide-circle-x', label: 'Error' },
  'abandoned': { color: 'neutral', icon: 'i-lucide-circle-slash', label: 'Abandoned' }
}

const defaultPresentation: CollectionStatusPresentation = {
  color: 'neutral',
  icon: 'i-lucide-circle',
  label: 'Unknown'
}

export function collectionStatusPresentation(status: string | null | undefined): CollectionStatusPresentation {
  const normalized = status?.trim().toLowerCase() ?? ''
  return statusPresentations[normalized] ?? {
    ...defaultPresentation,
    label: normalized ? normalized.charAt(0).toUpperCase() + normalized.slice(1) : defaultPresentation.label
  }
}

export function collectionStatusReasonLabel(reason: string | null | undefined): string {
  switch (reason) {
    case 'collection_created': return 'Collection created'
    case 'workflow_retried': return 'Workflow retried'
    case 'workflow_queued': return 'Workflow queued'
    case 'pipeline_acquired': return 'Pipeline acquired'
    case 'operator_decision_required': return 'Operator decision required'
    case 'operator_decision_received': return 'Operator decision received'
    case 'processing_resumed': return 'Processing resumed'
    case 'workflow_completed': return 'Workflow completed'
    case 'workflow_failed': return 'Workflow failed'
    case 'workflow_abandoned': return 'Workflow abandoned'
    case 'status_changed': return 'Status changed'
    default: return reason?.trim() || 'Status changed'
  }
}
