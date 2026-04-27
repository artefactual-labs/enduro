import type {
  EnduroCollectionWorkflowHistory,
  EnduroCollectionWorkflowStatus
} from '../openapi-generator'

export type EventTime = {
  seconds: number
  nanos: number
}

type EventTimeInput = EventTime | string | null | undefined

export type ParsedWorkflowActivity = {
  id: number
  attempts: number
  completedAt: string | null
  details: string
  durationSeconds: string | null
  isLocal: boolean
  name: string
  replayedAt: string | null
  startedAt: string | null
  status: string | null
}

export type ParsedWorkflowHistoryEvent = {
  activityName: string | null
  description: string
  eventTime: string | null
  id: number | null
  type: string
}

export type ParsedWorkflowStatus = {
  activities: ParsedWorkflowActivity[]
  activityError: boolean
  completedAt: string | null
  events: ParsedWorkflowHistoryEvent[]
  startedAt: string | null
  status: string
  workflowError: string
}

const ignoredActivities = new Set([
  'internalSessionCreationActivity',
  'internalSessionCompletionActivity'
])

function getRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' ? value as Record<string, unknown> : null
}

function getNestedValue(value: unknown, path: string[]): unknown {
  let current: unknown = value

  for (const key of path) {
    const record = getRecord(current)
    if (!record || !(key in record)) return undefined
    current = record[key]
  }

  return current
}

function normalizeEventTime(value: EventTimeInput): string | null {
  const date = eventTimeToDate(value)
  return date ? date.toISOString() : null
}

function getEventTime(details: unknown): string | null {
  return normalizeEventTime(getNestedValue(details, ['event_time']) as EventTimeInput)
}

function decodeBase64(input: string): string {
  try {
    return atob(input)
  } catch {
    return input
  }
}

function decodeStructuredPayload(input: string): unknown {
  const decoded = decodeBase64(input)

  try {
    return JSON.parse(decoded)
  } catch {
    return decoded
  }
}

function eventTimeToDate(value: EventTimeInput): Date | null {
  if (!value) return null

  if (typeof value === 'string') {
    const parsed = new Date(value)
    return Number.isNaN(parsed.getTime()) ? null : parsed
  }

  const record = getRecord(value)
  if (!record) return null

  const seconds = typeof record.seconds === 'number' ? record.seconds : Number(record.seconds)
  const nanos = typeof record.nanos === 'number' ? record.nanos : Number(record.nanos)

  if (!Number.isFinite(seconds) || !Number.isFinite(nanos)) return null

  return new Date(seconds * 1000 + nanos / 1_000_000)
}

function secondsBetween(startedAt: string | null, completedAt: string | null): string | null {
  const start = eventTimeToDate(startedAt)
  const end = eventTimeToDate(completedAt)
  if (!start || !end) return null

  const seconds = Math.max(0, (end.getTime() - start.getTime()) / 1000)
  return seconds.toLocaleString()
}

function workflowFailureDescription(failure: unknown): string {
  const record = getRecord(failure)
  if (!record) return ''

  let description = typeof record.message === 'string' ? record.message : ''
  const causeMessage = getNestedValue(record, ['cause', 'message'])
  if (typeof causeMessage === 'string' && causeMessage.length > 0) {
    description = description ? `${description}: ${causeMessage}` : causeMessage
  }

  return description
}

function stringifyForDisplay(value: unknown): string {
  if (typeof value === 'string') return value

  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return ''
  }
}

function getScheduledActivityName(details: unknown): string | null {
  const activityName = getNestedValue(details, ['Attributes', 'ActivityTaskScheduledEventAttributes', 'activity_type', 'name'])
  if (typeof activityName !== 'string' || ignoredActivities.has(activityName)) return null
  return activityName
}

function getScheduledEventId(details: unknown, path: string[]): number | null {
  const scheduledEventId = Number(getNestedValue(details, path))
  return Number.isFinite(scheduledEventId) ? scheduledEventId : null
}

function eventDescription(event: EnduroCollectionWorkflowHistory): string {
  const details = event.details

  if (event.type === 'ActivityTaskScheduled') {
    return ''
  }

  if (event.type === 'ActivityTaskFailed') {
    return stringifyForDisplay(getNestedValue(details, ['Attributes', 'ActivityTaskFailedEventAttributes']))
  }

  if (event.type === 'DecisionTaskScheduled') {
    const attemptValue = getNestedValue(details, ['decisionTaskScheduledEventAttributes', 'attempt'])
    const parsedAttempt = Number.parseInt(String(attemptValue ?? ''), 10)
    return Number.isFinite(parsedAttempt) ? `Attempts: ${parsedAttempt + 1}` : ''
  }

  if (event.type === 'WorkflowExecutionFailed') {
    return stringifyForDisplay(getNestedValue(details, ['Attributes', 'WorkflowExecutionFailedEventAttributes']))
  }

  if (event.type === 'WorkflowExecutionStarted') {
    return stringifyForDisplay(getNestedValue(details, ['Attributes', 'WorkflowExecutionStartedEventAttributes']))
  }

  return ''
}

export function parseWorkflowStatus(input: EnduroCollectionWorkflowStatus | null | undefined): ParsedWorkflowStatus {
  const activities = new Map<number, ParsedWorkflowActivity>()
  const activityNames = new Map<number, string>()
  const history = input?.history ?? []

  let startedAt: string | null = null
  let completedAt: string | null = null
  let workflowError = ''
  let activityError = false

  for (const event of history) {
    const details = event.details
    const eventId = typeof event.id === 'number' ? event.id : null
    const eventTime = getEventTime(details)

    if (event.type === 'MarkerRecorded') {
      const attrs = getNestedValue(details, ['Attributes', 'MarkerRecordedEventAttributes'])
      const markerName = getNestedValue(attrs, ['marker_name'])
      if (markerName !== 'LocalActivity') continue

      const payloads = getNestedValue(attrs, ['details', 'data', 'payloads'])
      const firstPayload = Array.isArray(payloads) ? getRecord(payloads[0]) : null
      if (!eventId || !firstPayload || typeof firstPayload.data !== 'string') continue

      const innerDetails = decodeStructuredPayload(firstPayload.data)
      const replayTime = getNestedValue(innerDetails, ['ReplayTime'])
      activities.set(eventId, {
        id: eventId,
        attempts: Number(getNestedValue(innerDetails, ['Attempt']) ?? 0),
        completedAt: null,
        details: '',
        durationSeconds: null,
        isLocal: true,
        name: String(getNestedValue(innerDetails, ['ActivityType']) ?? 'Local activity'),
        replayedAt: typeof replayTime === 'string' ? replayTime : null,
        startedAt: null,
        status: null
      })
      continue
    }

    if (event.type === 'ActivityTaskScheduled') {
      const attrs = getNestedValue(details, ['Attributes', 'ActivityTaskScheduledEventAttributes'])
      const name = getNestedValue(attrs, ['activity_type', 'name'])
      if (!eventId || typeof name !== 'string' || ignoredActivities.has(name)) continue

      activityNames.set(eventId, name)

      activities.set(eventId, {
        id: eventId,
        attempts: 0,
        completedAt: null,
        details: '',
        durationSeconds: null,
        isLocal: false,
        name,
        replayedAt: null,
        startedAt: eventTime,
        status: 'in progress'
      })
      continue
    }

    if (event.type === 'ActivityTaskStarted') {
      const attrs = getNestedValue(details, ['Attributes', 'ActivityTaskStartedEventAttributes'])
      const scheduledEventId = Number(getNestedValue(attrs, ['scheduled_event_id']))
      const activity = activities.get(scheduledEventId)
      if (!activity) continue

      activity.attempts = Number(getNestedValue(attrs, ['attempt']) ?? activity.attempts)
      continue
    }

    if (event.type === 'ActivityTaskFailed') {
      const attrs = getNestedValue(details, ['Attributes', 'ActivityTaskFailedEventAttributes'])
      const scheduledEventId = Number(getNestedValue(attrs, ['scheduled_event_id']))
      const activity = activities.get(scheduledEventId)
      activityError = true
      if (!activity) continue

      activity.status = 'error'
      activity.details = `Message: ${String(getNestedValue(attrs, ['failure', 'message']) ?? 'Unknown failure')}`
      activity.completedAt = eventTime
      activity.durationSeconds = secondsBetween(activity.startedAt, activity.completedAt)
      continue
    }

    if (event.type === 'ActivityTaskCompleted') {
      const attrs = getNestedValue(details, ['Attributes', 'ActivityTaskCompletedEventAttributes'])
      const scheduledEventId = Number(getNestedValue(attrs, ['scheduled_event_id']))
      const activity = activities.get(scheduledEventId)
      if (!activity) continue

      activity.status = 'done'
      activity.completedAt = eventTime
      activity.durationSeconds = secondsBetween(activity.startedAt, activity.completedAt)

      if (activity.name === 'async-completion-activity') {
        const result = getNestedValue(attrs, ['result'])
        if (typeof result === 'string' && result.length > 0) {
          activity.details = `User selection: ${decodeBase64(result)}.`
        }
      }
      continue
    }

    if (event.type === 'ActivityTaskTimedOut') {
      const attrs = getNestedValue(details, ['Attributes', 'ActivityTaskTimedOutEventAttributes'])
      const scheduledEventId = Number(getNestedValue(attrs, ['scheduled_event_id']))
      const activity = activities.get(scheduledEventId)
      if (!activity) continue

      activity.status = 'timed out'
      activity.details = `Timeout ${String(getNestedValue(attrs, ['timeout_type']) ?? 'unknown')}.`
      continue
    }

    if (event.type === 'WorkflowExecutionStarted') {
      startedAt = eventTime
      continue
    }

    if (event.type === 'WorkflowExecutionCompleted') {
      completedAt = eventTime
      continue
    }

    if (event.type === 'WorkflowExecutionFailed') {
      const attrs = getNestedValue(details, ['Attributes', 'WorkflowExecutionFailedEventAttributes'])
      workflowError = workflowFailureDescription(getNestedValue(attrs, ['failure']))
      completedAt = eventTime
    }
  }

  const parsedActivities = Array.from(activities.values()).sort((left, right) => right.id - left.id)
  const parsedEvents = history
    .slice()
    .reverse()
    .map((event): ParsedWorkflowHistoryEvent => {
      let activityName: string | null = null

      if (event.type === 'ActivityTaskScheduled') {
        activityName = getScheduledActivityName(event.details)
      } else if (event.type === 'ActivityTaskStarted') {
        const scheduledEventId = getScheduledEventId(event.details, ['Attributes', 'ActivityTaskStartedEventAttributes', 'scheduled_event_id'])
        activityName = scheduledEventId !== null ? activityNames.get(scheduledEventId) ?? null : null
      } else if (event.type === 'ActivityTaskCompleted') {
        const scheduledEventId = getScheduledEventId(event.details, ['Attributes', 'ActivityTaskCompletedEventAttributes', 'scheduled_event_id'])
        activityName = scheduledEventId !== null ? activityNames.get(scheduledEventId) ?? null : null
      } else if (event.type === 'ActivityTaskFailed') {
        const scheduledEventId = getScheduledEventId(event.details, ['Attributes', 'ActivityTaskFailedEventAttributes', 'scheduled_event_id'])
        activityName = scheduledEventId !== null ? activityNames.get(scheduledEventId) ?? null : null
      } else if (event.type === 'ActivityTaskTimedOut') {
        const scheduledEventId = getScheduledEventId(event.details, ['Attributes', 'ActivityTaskTimedOutEventAttributes', 'scheduled_event_id'])
        activityName = scheduledEventId !== null ? activityNames.get(scheduledEventId) ?? null : null
      }

      return {
        activityName,
        description: eventDescription(event),
        eventTime: getEventTime(event.details),
        id: typeof event.id === 'number' ? event.id : null,
        type: event.type || 'Unknown'
      }
    })

  return {
    activities: parsedActivities,
    activityError,
    completedAt,
    events: parsedEvents,
    startedAt,
    status: input?.status || 'unknown',
    workflowError
  }
}
