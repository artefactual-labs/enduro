# SwaggerApi

All URIs are relative to *http://localhost:9000*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**swaggerSwaggerSwaggerJson**](SwaggerApi.md#swaggerswaggerswaggerjson) | **GET** /swagger/swagger.json | Download internal/api/gen/http/openapi.json |



## swaggerSwaggerSwaggerJson

> swaggerSwaggerSwaggerJson()

Download internal/api/gen/http/openapi.json

JSON document containing the API swagger definition.

### Example

```ts
import {
  Configuration,
  SwaggerApi,
} from '';
import type { SwaggerSwaggerSwaggerJsonRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new SwaggerApi();

  try {
    const data = await api.swaggerSwaggerSwaggerJson();
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters

This endpoint does not need any parameter.

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | File downloaded |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

