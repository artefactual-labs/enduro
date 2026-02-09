
# ListResponseBody


## Properties

Name | Type
------------ | -------------
`items` | [Array&lt;EnduroStoredCollection&gt;](EnduroStoredCollection.md)
`nextCursor` | string

## Example

```typescript
import type { ListResponseBody } from ''

// TODO: Update the object below with actual values
const example = {
  "items": [{"aip_id":"d1845cb6-a5ea-474a-9ab8-26f9bcd919f5","completed_at":"1970-01-01T00:00:01Z","created_at":"1970-01-01T00:00:01Z","id":1,"name":"abc123","original_id":"abc123","pipeline_id":"d1845cb6-a5ea-474a-9ab8-26f9bcd919f5","run_id":"d1845cb6-a5ea-474a-9ab8-26f9bcd919f5","started_at":"1970-01-01T00:00:01Z","status":"in progress","transfer_id":"d1845cb6-a5ea-474a-9ab8-26f9bcd919f5","workflow_id":"d1845cb6-a5ea-474a-9ab8-26f9bcd919f5"}],
  "nextCursor": abc123,
} satisfies ListResponseBody

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ListResponseBody
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


