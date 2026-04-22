import type { BatchHintsResult, BatchStatusResult, EnduroStoredPipeline, SubmitRequestBody } from '~/openapi-generator'
import { ResponseError } from '~/openapi-generator/runtime'
import { useBatchPageData, useBatchStatusData } from '~/loaders/batch-page'
import { normalizeRouteStringParam } from '~/loaders/route-params'
import {
  buildBatchSubmitRequest,
  parseSavedBatchDefaults,
  type DefaultsMode
} from './useBatchImport.helpers'

type SelectOption = {
  label: string
  value: string
}

const batchDefaultsStorageKey = 'batchDefaults'
const statusRefreshDelayMs = 1000

const transferOptions: SelectOption[] = [
  { value: 'standard', label: 'Standard' },
  { value: 'zipfile', label: 'Zipfile' },
  { value: 'unzipped bag', label: 'Unzipped bag' },
  { value: 'zipped bag', label: 'Zipped bag' },
  { value: 'dspace', label: 'DSpace' },
  { value: 'maildir', label: 'Maildir' },
  { value: 'TRIM', label: 'TRIM' },
  { value: 'dataverse', label: 'Dataverse' }
]

const destinationModeOptions: SelectOption[] = [
  { value: 'completed-dir', label: 'Completed directory' },
  { value: 'retention-period', label: 'Retention period' }
]

function createDefaultStatus(): BatchStatusResult {
  return {
    running: false
  }
}

function createDefaultHints(): BatchHintsResult {
  return {
    completedDirs: []
  }
}

export function useBatchImport() {
  const enduroApi = useEnduroApi()
  const route = useRoute()
  const router = useRouter()
  const {
    data: pageData,
    isLoading: isLoadingPageData
  } = useBatchPageData()
  const {
    data: batchStatusData,
    isLoading: isLoadingStatus,
    reload: reloadStatus
  } = useBatchStatusData()

  const path = ref('')
  const selectedPipelineId = ref('')
  const selectedProcessingConfig = ref('')
  const selectedTransferType = ref('')
  const rejectDuplicates = ref(false)
  const excludeHiddenFiles = ref(false)
  const processNameMetadata = ref(false)
  const depth = ref(0)
  const destinationMode = ref<DefaultsMode>('completed-dir')
  const completedDir = ref('')
  const retentionPeriod = ref('')

  const isSubmitting = ref(false)
  const submitErrorMessage = ref('')
  const submitSuccessMessage = ref('')

  let statusTimer: number | null = null
  let isDisposed = false
  let suppressPipelineRouteSync = false

  const pipelines = computed(() => pageData.value?.pipelines ?? [])
  const processingOptions = computed(() => pageData.value?.processingOptions ?? [])
  const hints = computed(() => pageData.value?.hints ?? createDefaultHints())
  const status = computed(() => batchStatusData.value?.status ?? createDefaultStatus())

  const pipelinesErrorMessage = computed(() => pageData.value?.pipelinesErrorMessage ?? '')
  const processingErrorMessage = computed(() => pageData.value?.processingErrorMessage ?? '')
  const hintsErrorMessage = computed(() => pageData.value?.hintsErrorMessage ?? '')
  const statusErrorMessage = computed(() => batchStatusData.value?.errorMessage ?? '')

  const isLoadingPipelines = computed(() => isLoadingPageData.value)
  const isLoadingProcessing = computed(() => isLoadingPageData.value && Boolean(selectedPipelineId.value))
  const isLoadingHints = computed(() => isLoadingPageData.value)

  const pipelineOptions = computed<SelectOption[]>(() => (
    pipelines.value
      .filter((pipeline): pipeline is EnduroStoredPipeline & { id: string } => Boolean(pipeline.id))
      .map(pipeline => ({
        label: pipeline.name,
        value: pipeline.id
      }))
  ))

  const selectedPipeline = computed(() => (
    pipelines.value.find(pipeline => pipeline.id === selectedPipelineId.value) ?? null
  ))

  const isRunning = computed(() => status.value.running)
  const hasKnownCompletedDirs = computed(() => (hints.value.completedDirs?.length ?? 0) > 0)
  const canSubmit = computed(() => !isSubmitting.value && !isRunning.value && path.value.trim().length > 0)

  function clearStatusTimer() {
    if (!statusTimer) return
    window.clearTimeout(statusTimer)
    statusTimer = null
  }

  function scheduleStatusRefresh() {
    if (!import.meta.client || isDisposed || !isRunning.value) return
    clearStatusTimer()
    statusTimer = window.setTimeout(() => {
      if (isDisposed) return
      statusTimer = null
      void reloadStatus()
    }, statusRefreshDelayMs)
  }

  function saveDefaults() {
    if (!import.meta.client) return

    const defaults: SavedBatchDefaults = {
      completedDir: completedDir.value.trim(),
      depth: Number.isFinite(depth.value) ? Math.max(0, depth.value) : 0,
      excludeHiddenFiles: excludeHiddenFiles.value,
      mode: destinationMode.value,
      processNameMetadata: processNameMetadata.value,
      rejectDuplicates: rejectDuplicates.value,
      retentionPeriod: retentionPeriod.value.trim(),
      transferType: selectedTransferType.value
    }

    localStorage.setItem(batchDefaultsStorageKey, JSON.stringify(defaults))
  }

  function loadDefaults() {
    if (!import.meta.client) return

    const parsed = parseSavedBatchDefaults(localStorage.getItem(batchDefaultsStorageKey))
    if (!parsed) {
      localStorage.removeItem(batchDefaultsStorageKey)
      return
    }

    completedDir.value = parsed.completedDir
    depth.value = parsed.depth
    excludeHiddenFiles.value = parsed.excludeHiddenFiles
    processNameMetadata.value = parsed.processNameMetadata
    rejectDuplicates.value = parsed.rejectDuplicates
    retentionPeriod.value = parsed.retentionPeriod
    selectedTransferType.value = parsed.transferType
    destinationMode.value = parsed.mode
  }

  async function syncSelectedPipelineRoute(value: string) {
    const currentPipelineId = normalizeRouteStringParam(route.query.pipeline)
    if (currentPipelineId === value) return

    const nextQuery = { ...route.query }
    if (value) {
      nextQuery.pipeline = value
    } else {
      delete nextQuery.pipeline
    }

    await router.replace({ query: nextQuery })
  }

  async function loadStatus() {
    await reloadStatus()
  }

  function buildSubmitRequest(): SubmitRequestBody {
    return buildBatchSubmitRequest({
      completedDir: completedDir.value,
      depth: depth.value,
      destinationMode: destinationMode.value,
      excludeHiddenFiles: excludeHiddenFiles.value,
      path: path.value,
      pipelineName: selectedPipeline.value?.name ?? null,
      processNameMetadata: processNameMetadata.value,
      processingConfig: selectedProcessingConfig.value,
      rejectDuplicates: rejectDuplicates.value,
      retentionPeriod: retentionPeriod.value,
      transferType: selectedTransferType.value
    })
  }

  async function submit() {
    if (!canSubmit.value) return

    isSubmitting.value = true
    submitErrorMessage.value = ''
    submitSuccessMessage.value = ''

    try {
      await enduroApi.batches.submit(buildSubmitRequest())
      saveDefaults()
      submitSuccessMessage.value = 'Batch submitted.'
      path.value = ''
      await reloadStatus()
    } catch (error) {
      if (error instanceof ResponseError && error.response.status === 409) {
        submitErrorMessage.value = 'A batch started before this submission completed. Reloaded current status.'
        await reloadStatus()
      } else {
        submitErrorMessage.value = 'Could not submit the batch.'
      }
    } finally {
      isSubmitting.value = false
    }
  }

  function useCompletedDirHint(value: string) {
    completedDir.value = value
    destinationMode.value = 'completed-dir'
  }

  watch(() => pageData.value?.selectedPipelineId ?? normalizeRouteStringParam(route.query.pipeline), (value) => {
    suppressPipelineRouteSync = true
    selectedPipelineId.value = value
    suppressPipelineRouteSync = false
  }, { immediate: true })

  watch(selectedPipelineId, (value, previousValue) => {
    if (suppressPipelineRouteSync || value === previousValue) return
    selectedProcessingConfig.value = ''
    void syncSelectedPipelineRoute(value)
  })

  watch(isRunning, (running) => {
    if (running) {
      scheduleStatusRefresh()
    } else {
      clearStatusTimer()
    }
  }, { immediate: true })

  onMounted(() => {
    loadDefaults()
  })

  onBeforeUnmount(() => {
    isDisposed = true
    clearStatusTimer()
  })

  return {
    canSubmit,
    completedDir,
    destinationMode,
    destinationModeOptions,
    depth,
    excludeHiddenFiles,
    hasKnownCompletedDirs,
    hints,
    hintsErrorMessage,
    isLoadingHints,
    isLoadingPipelines,
    isLoadingProcessing,
    isLoadingStatus,
    isRunning,
    isSubmitting,
    path,
    pipelineOptions,
    pipelinesErrorMessage,
    processNameMetadata,
    processingErrorMessage,
    processingOptions,
    rejectDuplicates,
    retentionPeriod,
    selectedPipelineId,
    selectedProcessingConfig,
    selectedTransferType,
    status,
    statusErrorMessage,
    loadStatus,
    submit,
    submitErrorMessage,
    submitSuccessMessage,
    transferOptions,
    useCompletedDirHint
  }
}
