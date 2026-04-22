import { expect, test } from 'vitest'

import {
  isWorkflowExecutionTerminalStatus,
  workflowActivityStatusColor,
  workflowExecutionStatusColor
} from './workflow-summary'

test('workflow execution statuses preserve Temporal badge semantics', () => {
  expect(workflowExecutionStatusColor('COMPLETED')).toBe('success')
  expect(workflowExecutionStatusColor('RUNNING')).toBe('warning')
  expect(workflowExecutionStatusColor('FAILED')).toBe('error')
  expect(workflowExecutionStatusColor('Canceled')).toBe('error')
  expect(workflowExecutionStatusColor('TimedOut')).toBe('error')
  expect(workflowExecutionStatusColor('ContinuedAsNew')).toBe('warning')
  expect(workflowExecutionStatusColor('queued')).toBe('info')
  expect(workflowExecutionStatusColor('unknown')).toBe('neutral')
})

test('workflow terminal statuses stop auto-reload polling', () => {
  expect(isWorkflowExecutionTerminalStatus('COMPLETED')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('FAILED')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('TERMINATED')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('TIMED_OUT')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('Canceled')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('TimedOut')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('ContinuedAsNew')).toBe(true)
  expect(isWorkflowExecutionTerminalStatus('RUNNING')).toBe(false)
  expect(isWorkflowExecutionTerminalStatus('queued')).toBe(false)
})

test('workflow activity statuses preserve legacy collection badge semantics', () => {
  expect(workflowActivityStatusColor('done')).toBe('success')
  expect(workflowActivityStatusColor('in progress')).toBe('warning')
  expect(workflowActivityStatusColor('error')).toBe('error')
  expect(workflowActivityStatusColor('timed out')).toBe('error')
  expect(workflowActivityStatusColor('queued')).toBe('info')
  expect(workflowActivityStatusColor('unknown')).toBe('neutral')
  expect(workflowActivityStatusColor('continued_as_new')).toBe('warning')
})
