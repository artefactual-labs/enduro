
# SubmitRequestBody


## Properties

Name | Type
------------ | -------------
`completedDir` | string
`depth` | number
`excludeHiddenFiles` | boolean
`path` | string
`pipeline` | string
`processNameMetadata` | boolean
`processingConfig` | string
`rejectDuplicates` | boolean
`retentionPeriod` | string
`transferType` | string

## Example

```typescript
import type { SubmitRequestBody } from ''

// TODO: Update the object below with actual values
const example = {
  "completedDir": abc123,
  "depth": 1,
  "excludeHiddenFiles": false,
  "path": abc123,
  "pipeline": abc123,
  "processNameMetadata": false,
  "processingConfig": abc123,
  "rejectDuplicates": false,
  "retentionPeriod": abc123,
  "transferType": abc123,
} satisfies SubmitRequestBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SubmitRequestBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


