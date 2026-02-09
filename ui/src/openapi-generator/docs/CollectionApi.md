# CollectionApi

All URIs are relative to *http://localhost:9000*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**collectionBulk**](CollectionApi.md#collectionbulk) | **POST** /collection/bulk | bulk collection |
| [**collectionBulkStatus**](CollectionApi.md#collectionbulkstatus) | **GET** /collection/bulk | bulk_status collection |
| [**collectionCancel**](CollectionApi.md#collectioncancel) | **POST** /collection/{id}/cancel | cancel collection |
| [**collectionDecide**](CollectionApi.md#collectiondecideoperation) | **POST** /collection/{id}/decision | decide collection |
| [**collectionDelete**](CollectionApi.md#collectiondelete) | **DELETE** /collection/{id} | delete collection |
| [**collectionDownload**](CollectionApi.md#collectiondownload) | **GET** /collection/{id}/download | download collection |
| [**collectionList**](CollectionApi.md#collectionlist) | **GET** /collection | list collection |
| [**collectionMonitor**](CollectionApi.md#collectionmonitor) | **GET** /collection/monitor | monitor collection |
| [**collectionRetry**](CollectionApi.md#collectionretry) | **POST** /collection/{id}/retry | retry collection |
| [**collectionShow**](CollectionApi.md#collectionshow) | **GET** /collection/{id} | show collection |
| [**collectionWorkflow**](CollectionApi.md#collectionworkflow) | **GET** /collection/{id}/workflow | workflow collection |



## collectionBulk

> BulkResult collectionBulk(bulkRequestBody)

bulk collection

Bulk operations (retry, cancel...).

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionBulkRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // BulkRequestBody
    bulkRequestBody: {"operation":"cancel","size":1,"status":"in progress"},
  } satisfies CollectionBulkRequest;

  try {
    const data = await api.collectionBulk(body);
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
| **bulkRequestBody** | [BulkRequestBody](BulkRequestBody.md) |  | |

### Return type

[**BulkResult**](BulkResult.md)

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


## collectionBulkStatus

> BulkStatusResult collectionBulkStatus()

bulk_status collection

Retrieve status of current bulk operation.

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionBulkStatusRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  try {
    const data = await api.collectionBulkStatus();
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

[**BulkStatusResult**](BulkStatusResult.md)

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


## collectionCancel

> collectionCancel(id)

cancel collection

Cancel collection processing by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionCancelRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to remove
    id: 1,
  } satisfies CollectionCancelRequest;

  try {
    const data = await api.collectionCancel(body);
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
| **id** | `number` | Identifier of collection to remove | [Defaults to `undefined`] |

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/vnd.goa.error`, `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **400** | not_running: Bad Request response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionDecide

> collectionDecide(id, collectionDecideRequest)

decide collection

Make decision for a pending collection by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionDecideOperationRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to look up
    id: 1,
    // CollectionDecideRequest
    collectionDecideRequest: {"option":"abc123"},
  } satisfies CollectionDecideOperationRequest;

  try {
    const data = await api.collectionDecide(body);
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
| **id** | `number` | Identifier of collection to look up | [Defaults to `undefined`] |
| **collectionDecideRequest** | [CollectionDecideRequest](CollectionDecideRequest.md) |  | |

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: `application/json`
- **Accept**: `application/vnd.goa.error`, `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **400** | not_valid: Bad Request response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionDelete

> collectionDelete(id)

delete collection

Delete collection by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionDeleteRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to delete
    id: 1,
  } satisfies CollectionDeleteRequest;

  try {
    const data = await api.collectionDelete(body);
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
| **id** | `number` | Identifier of collection to delete | [Defaults to `undefined`] |

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | No Content response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionDownload

> Blob collectionDownload(id)

download collection

Download collection by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionDownloadRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to look up
    id: 1,
  } satisfies CollectionDownloadRequest;

  try {
    const data = await api.collectionDownload(body);
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
| **id** | `number` | Identifier of collection to look up | [Defaults to `undefined`] |

### Return type

**Blob**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  * Content-Disposition -  <br>  * Content-Length -  <br>  * Content-Type -  <br>  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionList

> ListResponseBody collectionList(name, originalId, transferId, aipId, pipelineId, earliestCreatedTime, latestCreatedTime, status, cursor)

list collection

List all stored collections

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionListRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // string (optional)
    name: abc123,
    // string (optional)
    originalId: abc123,
    // string | Identifier of Archivematica tranfser (optional)
    transferId: d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
    // string | Identifier of Archivematica AIP (optional)
    aipId: d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
    // string | Identifier of Archivematica pipeline (optional)
    pipelineId: d1845cb6-a5ea-474a-9ab8-26f9bcd919f5,
    // Date (optional)
    earliestCreatedTime: e1d563b0-1474-4155-beed-f2d3a12e1529,
    // Date (optional)
    latestCreatedTime: e1d563b0-1474-4155-beed-f2d3a12e1529,
    // 'new' | 'in progress' | 'done' | 'error' | 'unknown' | 'queued' | 'pending' | 'abandoned' (optional)
    status: in progress,
    // string | Pagination cursor (optional)
    cursor: abc123,
  } satisfies CollectionListRequest;

  try {
    const data = await api.collectionList(body);
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
| **originalId** | `string` |  | [Optional] [Defaults to `undefined`] |
| **transferId** | `string` | Identifier of Archivematica tranfser | [Optional] [Defaults to `undefined`] |
| **aipId** | `string` | Identifier of Archivematica AIP | [Optional] [Defaults to `undefined`] |
| **pipelineId** | `string` | Identifier of Archivematica pipeline | [Optional] [Defaults to `undefined`] |
| **earliestCreatedTime** | `Date` |  | [Optional] [Defaults to `undefined`] |
| **latestCreatedTime** | `Date` |  | [Optional] [Defaults to `undefined`] |
| **status** | `new`, `in progress`, `done`, `error`, `unknown`, `queued`, `pending`, `abandoned` |  | [Optional] [Defaults to `undefined`] [Enum: new, in progress, done, error, unknown, queued, pending, abandoned] |
| **cursor** | `string` | Pagination cursor | [Optional] [Defaults to `undefined`] |

### Return type

[**ListResponseBody**](ListResponseBody.md)

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


## collectionMonitor

> collectionMonitor()

monitor collection

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionMonitorRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  try {
    const data = await api.collectionMonitor();
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
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **101** | Switching Protocols response. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionRetry

> collectionRetry(id)

retry collection

Retry collection processing by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionRetryRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to retry
    id: 1,
  } satisfies CollectionRetryRequest;

  try {
    const data = await api.collectionRetry(body);
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
| **id** | `number` | Identifier of collection to retry | [Defaults to `undefined`] |

### Return type

`void` (Empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/vnd.goa.error`, `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **400** | not_running: Bad Request response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionShow

> EnduroStoredCollection collectionShow(id)

show collection

Show collection by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionShowRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to show
    id: 1,
  } satisfies CollectionShowRequest;

  try {
    const data = await api.collectionShow(body);
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
| **id** | `number` | Identifier of collection to show | [Defaults to `undefined`] |

### Return type

[**EnduroStoredCollection**](EnduroStoredCollection.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


## collectionWorkflow

> EnduroCollectionWorkflowStatus collectionWorkflow(id)

workflow collection

Retrieve workflow status by ID

### Example

```ts
import {
  Configuration,
  CollectionApi,
} from '';
import type { CollectionWorkflowRequest } from '';

async function example() {
  console.log("🚀 Testing  SDK...");
  const api = new CollectionApi();

  const body = {
    // number | Identifier of collection to look up
    id: 1,
  } satisfies CollectionWorkflowRequest;

  try {
    const data = await api.collectionWorkflow(body);
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
| **id** | `number` | Identifier of collection to look up | [Defaults to `undefined`] |

### Return type

[**EnduroCollectionWorkflowStatus**](EnduroCollectionWorkflowStatus.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: `application/json`


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | OK response. |  -  |
| **404** | not_found: Collection not found |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)

