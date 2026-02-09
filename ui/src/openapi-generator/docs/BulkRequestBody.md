
# BulkRequestBody


## Properties

Name | Type
------------ | -------------
`operation` | string
`size` | number
`status` | string

## Example

```typescript
import type { BulkRequestBody } from ''

// TODO: Update the object below with actual values
const example = {
  "operation": cancel,
  "size": 1,
  "status": in progress,
} satisfies BulkRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as BulkRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


