
# EnduroCollectionWorkflowStatus

WorkflowStatus describes the processing workflow status of a collection.

## Properties

Name | Type
------------ | -------------
`history` | [Array&lt;EnduroCollectionWorkflowHistory&gt;](EnduroCollectionWorkflowHistory.md)
`status` | string

## Example

```typescript
import type { EnduroCollectionWorkflowStatus } from ''

// TODO: Update the object below with actual values
const example = {
  "history": [{"details":"abc123","id":1,"type":"abc123"}],
  "status": abc123,
} satisfies EnduroCollectionWorkflowStatus

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as EnduroCollectionWorkflowStatus
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


