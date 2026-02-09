
# Collection

Collection describes a collection to be stored.

## Properties

Name | Type
------------ | -------------
`aipId` | string
`completedAt` | Date
`createdAt` | Date
`name` | string
`originalId` | string
`pipelineId` | string
`runId` | string
`startedAt` | Date
`status` | string
`transferId` | string
`workflowId` | string

## Example

```typescript
import type { Collection } from ''

// TODO: Update the object below with actual values
const example = {
  "aipId": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  "completedAt": 1970-01-01T00:00:01Z,
  "createdAt": 1970-01-01T00:00:01Z,
  "name": abc123,
  "originalId": abc123,
  "pipelineId": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  "runId": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  "startedAt": 1970-01-01T00:00:01Z,
  "status": in progress,
  "transferId": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  "workflowId": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
} satisfies Collection

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as Collection
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


