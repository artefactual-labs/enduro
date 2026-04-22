type WorkflowStatusColor = 'success' | 'warning' | 'error' | 'neutral' | 'info'

function normalizeStatus(status: string | null | undefined): string {
  const normalized = String(status ?? '').trim().toLowerCase()

  if (normalized === 'timedout' || normalized === 'timed out') return 'timed_out'
  if (normalized === 'continuedasnew' || normalized === 'continued as new') return 'continued_as_new'
  if (normalized === 'cancelled') return 'canceled'

  return normalized
}

export function isWorkflowExecutionTerminalStatus(status: string | null | undefined): boolean {
  const normalized = normalizeStatus(status)

  return normalized === 'completed'
    || normalized === 'failed'
    || normalized === 'terminated'
    || normalized === 'timed_out'
    || normalized === 'canceled'
    || normalized === 'continued_as_new'
}

export function workflowExecutionStatusColor(status: string | null | undefined): WorkflowStatusColor {
  const normalized = normalizeStatus(status)

  if (normalized === 'completed') return 'success'
  if (normalized === 'failed' || normalized === 'terminated' || normalized === 'timed_out' || normalized === 'canceled') return 'error'
  if (normalized === 'running' || normalized === 'active' || normalized === 'continued_as_new') return 'warning'
  if (normalized === 'pending' || normalized === 'queued') return 'info'

  return 'neutral'
}

export function workflowActivityStatusColor(status: string | null | undefined): WorkflowStatusColor {
  const normalized = normalizeStatus(status)

  if (normalized === 'done') return 'success'
  if (normalized === 'in progress') return 'warning'
  if (normalized === 'error' || normalized === 'timed out') return 'error'
  if (normalized === 'queued' || normalized === 'pending') return 'info'
  if (normalized === 'new' || normalized === 'unknown' || normalized === 'abandoned') return 'neutral'

  return workflowExecutionStatusColor(status)
}
