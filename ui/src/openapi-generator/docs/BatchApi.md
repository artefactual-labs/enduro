# BatchApi

All URIs are relative to *http://localhost:9000*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**batchHints**](BatchApi.md#batchhints) | **GET** /batch/hints | hints batch |
| [**batchStatus**](BatchApi.md#batchstatus) | **GET** /batch | status batch |
| [**batchSubmit**](BatchApi.md#batchsubmit) | **POST** /batch | submit batch |



## batchHints

> BatchHintsResult batchHints()

hints batch

Retrieve form hints

### Example

```ts
import {
  Configuration,
  BatchApi,
} from '';
import type { BatchHintsRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new BatchApi();

  try {
    const data = await api.batchHints();
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

[**BatchHintsResult**](BatchHintsResult.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## batchStatus

> BatchStatusResult batchStatus()

status batch

Retrieve status of current batch operation.

### Example

```ts
import {
  Configuration,
  BatchApi,
} from '';
import type { BatchStatusRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new BatchApi();

  try {
    const data = await api.batchStatus();
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

[**BatchStatusResult**](BatchStatusResult.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## batchSubmit

> BatchResult batchSubmit(submitRequestBody)

submit batch

Submit a new batch

### Example

```ts
import {
  Configuration,
  BatchApi,
} from '';
import type { BatchSubmitRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new BatchApi();

  const body = {
    // SubmitRequestBody
    submitRequestBody: {"completed_dir":"abc123","depth":1,"exclude_hidden_files":false,"path":"abc123","pipeline":"abc123","process_name_metadata":false,"processing_config":"abc123","reject_duplicates":false,"retention_period":"abc123","transfer_type":"abc123"},
  } satisfies BatchSubmitRequest;

  try {
    const data = await api.batchSubmit(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```

### Parameters


| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **submitRequestBody** | [SubmitRequestBody](SubmitRequestBody.md) |  | |

### Return type

[**BatchResult**](BatchResult.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/json`, `application/vnd.goa.error`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **202** | Accepted response. |  -  |
| **400** | not_valid: Bad Request response. |  -  |
| **409** | not_available: Conflict response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

