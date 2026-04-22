import {
  BatchApi,
  CollectionApi,
  Configuration,
  PipelineApi,
  type BatchHintsResult,
  type BatchResult,
  type BatchStatusResult,
  type BulkRequestBody,
  type BulkResult,
  type BulkStatusResult,
  type CollectionApiCollectionListRequest,
  type EnduroCollectionWorkflowStatus,
  type EnduroDetailedStoredCollection,
  type EnduroStoredPipeline,
  type InitOverrideFunction,
  type ListResponseBody,
  type PipelineApiPipelineListRequest,
  type RetryResult,
  type SubmitRequestBody
} from '~/openapi-generator'

declare global {
  interface Window {
    fgt_sslvpn?: {
      url_rewrite: (url: string) => string
    }
  }
}

type RuntimeConfigLike = {
  public?: {
    enduroApiBase?: string
  }
  app?: {
    baseURL?: string
  }
}

type RequestOptions = RequestInit | InitOverrideFunction

export type EnduroApiClient = {
  monitorUrl: string
  collections: {
    list: (request?: CollectionApiCollectionListRequest, requestOptions?: RequestOptions) => Promise<ListResponseBody>
    show: (id: number, requestOptions?: RequestOptions) => Promise<EnduroDetailedStoredCollection>
    workflow: (id: number, requestOptions?: RequestOptions) => Promise<EnduroCollectionWorkflowStatus>
    retry: (id: number, requestOptions?: RequestOptions) => Promise<RetryResult>
    cancel: (id: number, requestOptions?: RequestOptions) => Promise<void>
    remove: (id: number, requestOptions?: RequestOptions) => Promise<void>
    decide: (id: number, option: string, requestOptions?: RequestOptions) => Promise<void>
    bulk: (request: BulkRequestBody, requestOptions?: RequestOptions) => Promise<BulkResult>
    bulkStatus: (requestOptions?: RequestOptions) => Promise<BulkStatusResult>
  }
  pipelines: {
    list: (request?: PipelineApiPipelineListRequest, requestOptions?: RequestOptions) => Promise<EnduroStoredPipeline[]>
    show: (id: string, requestOptions?: RequestOptions) => Promise<EnduroStoredPipeline>
    processing: (id: string, requestOptions?: RequestOptions) => Promise<string[]>
  }
  batches: {
    hints: (requestOptions?: RequestOptions) => Promise<BatchHintsResult>
    status: (requestOptions?: RequestOptions) => Promise<BatchStatusResult>
    submit: (request: SubmitRequestBody, requestOptions?: RequestOptions) => Promise<BatchResult>
  }
  system: {
    versionHeader: (requestOptions?: RequestOptions) => Promise<string | null>
  }
}

function trimTrailingSlash(value: string): string {
  if (!value) return ''
  return value.replace(/\/+$/, '')
}

function joinBasePath(basePath: string, endpoint: string): string {
  if (!basePath) return endpoint
  return `${basePath}${endpoint}`
}

function maybeRewriteForSslVpn(path: string): string {
  if (!import.meta.client) return path

  const rewriteFn = window.fgt_sslvpn?.url_rewrite
  if (typeof rewriteFn !== 'function') return path

  try {
    return trimTrailingSlash(rewriteFn(path))
  } catch {
    return path
  }
}

function browserOrigin(): string {
  if (!import.meta.client) return ''

  const { location } = window
  return trimTrailingSlash(`${location.protocol}//${location.hostname}${location.port ? `:${location.port}` : ''}`)
}

function resolveConfiguredBasePath(runtimeConfig: RuntimeConfigLike): string {
  let configured = trimTrailingSlash(String(runtimeConfig.public?.enduroApiBase ?? ''))
  const appBase = trimTrailingSlash(String(runtimeConfig.app?.baseURL ?? ''))

  if (configured && appBase && configured === appBase) {
    configured = ''
  }

  if (configured.startsWith('/') && appBase && configured.startsWith(`${appBase}/`)) {
    configured = trimTrailingSlash(configured.slice(appBase.length))
  }

  return configured
}

export function resolveEnduroApiBasePath(runtimeConfig: RuntimeConfigLike): string {
  const configured = resolveConfiguredBasePath(runtimeConfig)
  if (!import.meta.client) return configured

  const candidate = configured
    ? trimTrailingSlash(new URL(configured, browserOrigin() || window.location.origin).toString())
    : browserOrigin()

  return maybeRewriteForSslVpn(candidate)
}

export function createEnduroApiClient(basePath: string): EnduroApiClient {
  const configuration = new Configuration({ basePath })
  const collection = new CollectionApi(configuration)
  const pipeline = new PipelineApi(configuration)
  const batch = new BatchApi(configuration)

  return {
    monitorUrl: joinBasePath(basePath, '/collection/monitor'),
    collections: {
      list: (request = {}, requestOptions) => collection.collectionList(request, requestOptions),
      show: (id, requestOptions) => collection.collectionShow({ id }, requestOptions),
      workflow: (id, requestOptions) => collection.collectionWorkflow({ id }, requestOptions),
      retry: (id, requestOptions) => collection.collectionRetry({ id }, requestOptions),
      cancel: (id, requestOptions) => collection.collectionCancel({ id }, requestOptions),
      remove: (id, requestOptions) => collection.collectionDelete({ id }, requestOptions),
      decide: (id, option, requestOptions) => collection.collectionDecide({
        id,
        collectionDecideRequest: { option }
      }, requestOptions),
      bulk: (request, requestOptions) => collection.collectionBulk({ bulkRequestBody: request }, requestOptions),
      bulkStatus: requestOptions => collection.collectionBulkStatus(requestOptions)
    },
    pipelines: {
      list: (request = {}, requestOptions) => pipeline.pipelineList(request, requestOptions),
      show: (id, requestOptions) => pipeline.pipelineShow({ id }, requestOptions),
      processing: (id, requestOptions) => pipeline.pipelineProcessing({ id }, requestOptions)
    },
    batches: {
      hints: requestOptions => batch.batchHints(requestOptions),
      status: requestOptions => batch.batchStatus(requestOptions),
      submit: (request, requestOptions) => batch.batchSubmit({ submitRequestBody: request }, requestOptions)
    },
    system: {
      async versionHeader(requestOptions): Promise<string | null> {
        const response = await collection.collectionListRaw({ cursor: '0' }, requestOptions)
        return response.raw.headers.get('x-enduro-version')?.trim() ?? null
      }
    }
  }
}
