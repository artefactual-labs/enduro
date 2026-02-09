
# ModelError


## Properties

Name | Type
------------ | -------------
`fault` | boolean
`id` | string
`message` | string
`name` | string
`temporary` | boolean
`timeout` | boolean

## Example

```typescript
import type { ModelError } from ''

// TODO: Update the object below with actual values
const example = {
  "fault": false,
  "id": 123abc,
  "message": parameter 'p' must be an integer,
  "name": bad_request,
  "temporary": false,
  "timeout": false,
} satisfies ModelError

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ModelError
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


