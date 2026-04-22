import { expect, test } from 'vitest'

import { parseWorkflowStatus } from './workflow-history'

function buildEventTime(value: string) {
  const date = new Date(value)
  return {
    seconds: Math.floor(date.getTime() / 1000),
    nanos: (date.getTime() % 1000) * 1_000_000
  }
}

test('parseWorkflowStatus leaves local activity status unset', () => {
  const payload = Buffer.from(JSON.stringify({
    ActivityType: 'updatePackageLocalActivity',
    Attempt: 1,
    ReplayTime: '2026-04-22T18:47:00Z'
  })).toString('base64')

  const parsed = parseWorkflowStatus({
    history: [
      {
        id: 17,
        type: 'MarkerRecorded',
        details: {
          Attributes: {
            MarkerRecordedEventAttributes: {
              marker_name: 'LocalActivity',
              details: {
                data: {
                  payloads: [{ data: payload }]
                }
              }
            }
          }
        }
      }
    ],
    status: 'running'
  })

  expect(parsed.activities.length).toBe(1)
  expect(parsed.activities[0]?.name).toBe('updatePackageLocalActivity')
  expect(parsed.activities[0]?.status).toBeNull()
  expect(parsed.activities[0]?.replayedAt).toBe('2026-04-22T18:47:00Z')
})

test('parseWorkflowStatus tracks scheduled activities through completion and async user selection', () => {
  const parsed = parseWorkflowStatus({
    history: [
      {
        id: 1,
        type: 'WorkflowExecutionStarted',
        details: { event_time: buildEventTime('2026-04-22T18:47:00Z') }
      },
      {
        id: 2,
        type: 'ActivityTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T18:47:05Z'),
          Attributes: {
            ActivityTaskScheduledEventAttributes: {
              activity_type: { name: 'async-completion-activity' }
            }
          }
        }
      },
      {
        id: 3,
        type: 'ActivityTaskStarted',
        details: {
          Attributes: {
            ActivityTaskStartedEventAttributes: {
              scheduled_event_id: 2,
              attempt: 2
            }
          }
        }
      },
      {
        id: 4,
        type: 'ActivityTaskCompleted',
        details: {
          event_time: buildEventTime('2026-04-22T18:47:12Z'),
          Attributes: {
            ActivityTaskCompletedEventAttributes: {
              scheduled_event_id: 2,
              result: Buffer.from('approve').toString('base64')
            }
          }
        }
      },
      {
        id: 5,
        type: 'WorkflowExecutionCompleted',
        details: { event_time: buildEventTime('2026-04-22T18:47:30Z') }
      }
    ],
    status: 'completed'
  })

  expect(parsed.startedAt).toBe('2026-04-22T18:47:00.000Z')
  expect(parsed.completedAt).toBe('2026-04-22T18:47:30.000Z')
  expect(parsed.activities.length).toBe(1)
  expect(parsed.activities[0]?.status).toBe('done')
  expect(parsed.activities[0]?.attempts).toBe(2)
  expect(parsed.activities[0]?.durationSeconds).toBe('7')
  expect(parsed.activities[0]?.details).toBe('User selection: approve.')
  expect(parsed.events[1]?.type).toBe('ActivityTaskCompleted')
  expect(parsed.events[1]?.activityName).toBe('async-completion-activity')
  expect(parsed.events[2]?.type).toBe('ActivityTaskStarted')
  expect(parsed.events[2]?.activityName).toBe('async-completion-activity')
  expect(parsed.events[3]?.type).toBe('ActivityTaskScheduled')
  expect(parsed.events[3]?.activityName).toBe('async-completion-activity')
  expect(parsed.events[3]?.description).toBe('')
})

test('parseWorkflowStatus reports activity failures, timeouts, and workflow failures', () => {
  const parsed = parseWorkflowStatus({
    history: [
      {
        id: 9,
        type: 'ActivityTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T18:50:00Z'),
          Attributes: {
            ActivityTaskScheduledEventAttributes: {
              activity_type: { name: 'download-activity' }
            }
          }
        }
      },
      {
        id: 10,
        type: 'ActivityTaskFailed',
        details: {
          event_time: buildEventTime('2026-04-22T18:50:04Z'),
          Attributes: {
            ActivityTaskFailedEventAttributes: {
              scheduled_event_id: 9,
              failure: { message: 'download failed' }
            }
          }
        }
      },
      {
        id: 11,
        type: 'ActivityTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T18:51:00Z'),
          Attributes: {
            ActivityTaskScheduledEventAttributes: {
              activity_type: { name: 'poll-ingest-activity' }
            }
          }
        }
      },
      {
        id: 12,
        type: 'ActivityTaskTimedOut',
        details: {
          Attributes: {
            ActivityTaskTimedOutEventAttributes: {
              scheduled_event_id: 11,
              timeout_type: 'START_TO_CLOSE'
            }
          }
        }
      },
      {
        id: 13,
        type: 'WorkflowExecutionFailed',
        details: {
          event_time: buildEventTime('2026-04-22T18:52:00Z'),
          Attributes: {
            WorkflowExecutionFailedEventAttributes: {
              failure: {
                message: 'workflow failed',
                cause: {
                  message: 'pipeline unavailable'
                }
              }
            }
          }
        }
      }
    ],
    status: 'failed'
  })

  expect(parsed.activityError).toBe(true)
  expect(parsed.workflowError).toBe('workflow failed: pipeline unavailable')
  expect(parsed.completedAt).toBe('2026-04-22T18:52:00.000Z')
  expect(parsed.activities[0]?.status).toBe('error')
  expect(parsed.activities[0]?.details).toBe('Message: download failed')
  expect(parsed.activities[1]?.status).toBe('timed out')
  expect(parsed.activities[1]?.details).toBe('Timeout START_TO_CLOSE.')

  const timedOutEvent = parsed.events.find(event => event.type === 'ActivityTaskTimedOut')
  const failedEvent = parsed.events.find(event => event.type === 'ActivityTaskFailed')

  expect(timedOutEvent?.activityName).toBe('poll-ingest-activity')
  expect(failedEvent?.activityName).toBe('download-activity')
})

test('parseWorkflowStatus ignores internal activities and builds reversed event history descriptions', () => {
  const parsed = parseWorkflowStatus({
    history: [
      {
        id: 21,
        type: 'ActivityTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T19:00:00Z'),
          Attributes: {
            ActivityTaskScheduledEventAttributes: {
              activity_type: { name: 'internalSessionCreationActivity' }
            }
          }
        }
      },
      {
        id: 22,
        type: 'DecisionTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T19:00:01Z'),
          decisionTaskScheduledEventAttributes: {
            attempt: 1
          }
        }
      },
      {
        id: 23,
        type: 'WorkflowExecutionStarted',
        details: {
          event_time: buildEventTime('2026-04-22T19:00:02Z'),
          Attributes: {
            WorkflowExecutionStartedEventAttributes: {
              parent_workflow_execution: { workflow_id: 'parent' }
            }
          }
        }
      }
    ],
    status: 'running'
  })

  expect(parsed.activities.length).toBe(0)
  expect(parsed.events[0]?.type).toBe('WorkflowExecutionStarted')
  expect(parsed.events[0]?.description ?? '').toMatch(/parent_workflow_execution/)
  expect(parsed.events[1]?.description).toBe('Attempts: 2')
})

test('parseWorkflowStatus does not expose ignored internal activity names in history events', () => {
  const parsed = parseWorkflowStatus({
    history: [
      {
        id: 30,
        type: 'ActivityTaskScheduled',
        details: {
          event_time: buildEventTime('2026-04-22T19:10:00Z'),
          Attributes: {
            ActivityTaskScheduledEventAttributes: {
              activity_type: { name: 'internalSessionCreationActivity' }
            }
          }
        }
      }
    ],
    status: 'running'
  })

  expect(parsed.events[0]?.type).toBe('ActivityTaskScheduled')
  expect(parsed.events[0]?.activityName).toBeNull()
})
