# PipelineApi

All URIs are relative to *http://localhost:9000*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**pipelineList**](PipelineApi.md#pipelinelist) | **GET** /pipeline | list pipeline |
| [**pipelineProcessing**](PipelineApi.md#pipelineprocessing) | **GET** /pipeline/{id}/processing | processing pipeline |
| [**pipelineShow**](PipelineApi.md#pipelineshow) | **GET** /pipeline/{id} | show pipeline |



## pipelineList

> Array&lt;EnduroStoredPipeline&gt; pipelineList(name, status)

list pipeline

List all known pipelines

### Example

```ts
import {
  Configuration,
  PipelineApi,
} from '';
import type { PipelineListRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new PipelineApi();

  const body = {
    // string (optional)
    name: abc123,
    // boolean (optional)
    status: false,
  } satisfies PipelineListRequest;

  try {
    const data = await api.pipelineList(body);
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
| **name** | `string` |  | [Optional] [Defaults to `undefined`] |
| **status** | `boolean` |  | [Optional] [Defaults to `false`] |

### Return type

[**Array&lt;EnduroStoredPipeline&gt;**](EnduroStoredPipeline.md)

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


## pipelineProcessing

> Array&lt;string&gt; pipelineProcessing(id)

processing pipeline

List all processing configurations of a pipeline given its ID

### Example

```ts
import {
  Configuration,
  PipelineApi,
} from '';
import type { PipelineProcessingRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new PipelineApi();

  const body = {
    // string | Identifier of pipeline
    id: d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  } satisfies PipelineProcessingRequest;

  try {
    const data = await api.pipelineProcessing(body);
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
| **id** | `string` | Identifier of pipeline | [Defaults to `undefined`] |

### Return type

**Array<string>**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **404** | not_found: Pipeline not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## pipelineShow

> EnduroStoredPipeline pipelineShow(id)

show pipeline

Show pipeline by ID

### Example

```ts
import {
  Configuration,
  PipelineApi,
} from '';
import type { PipelineShowRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new PipelineApi();

  const body = {
    // string | Identifier of pipeline to show
    id: d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
  } satisfies PipelineShowRequest;

  try {
    const data = await api.pipelineShow(body);
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
| **id** | `string` | Identifier of pipeline to show | [Defaults to `undefined`] |

### Return type

[**EnduroStoredPipeline**](EnduroStoredPipeline.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **404** | not_found: Pipeline not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

