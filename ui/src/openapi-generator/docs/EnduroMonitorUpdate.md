
# EnduroMonitorUpdate


## Properties

Name | Type
------------ | -------------
`id` | number
`item` | [EnduroStoredCollection](EnduroStoredCollection.md)
`timestamp` | Date
`type` | string

## Example

```typescript
import type { EnduroMonitorUpdate } from ''

// TODO: Update the object below with actual values
const example = {
  "id": 1,
  "item": null,
  "timestamp": 1970-01-01T00:00:01Z,
  "type": abc123,
} satisfies EnduroMonitorUpdate

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as EnduroMonitorUpdate
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


