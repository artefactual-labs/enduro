
# BulkStatusResult


## Properties

Name | Type
------------ | -------------
`closedAt` | Date
`runId` | string
`running` | boolean
`startedAt` | Date
`status` | string
`workflowId` | string

## Example

```typescript
import type { BulkStatusResult } from ''

// TODO: Update the object below with actual values
const example = {
  "closedAt": 1970-01-01T00:00:01Z,
  "runId": abc123,
  "running": false,
  "startedAt": 1970-01-01T00:00:01Z,
  "status": abc123,
  "workflowId": abc123,
} satisfies BulkStatusResult

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as BulkStatusResult
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


