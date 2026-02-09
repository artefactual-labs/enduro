
# BatchStatusResult


## Properties

Name | Type
------------ | -------------
`runId` | string
`running` | boolean
`status` | string
`workflowId` | string

## Example

```typescript
import type { BatchStatusResult } from ''

// TODO: Update the object below with actual values
const example = {
  "runId": abc123,
  "running": false,
  "status": abc123,
  "workflowId": abc123,
} satisfies BatchStatusResult

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as BatchStatusResult
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


