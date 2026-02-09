
# EnduroStoredPipeline

StoredPipeline describes a pipeline retrieved by this service.

## Properties

Name | Type
------------ | -------------
`capacity` | number
`current` | number
`id` | string
`name` | string
`status` | string

## Example

```typescript
import type { EnduroStoredPipeline } from ''

// TODO: Update the object below with actual values
const example = {
  "capacity": 1,
  "current": 1,
  "id": d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  "name": abc123,
  "status": abc123,
} satisfies EnduroStoredPipeline

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as EnduroStoredPipeline
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


