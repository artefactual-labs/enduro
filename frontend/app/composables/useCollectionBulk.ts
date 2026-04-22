import type { BulkRequestBody } from '~/openapi-generator'
import {
  BulkRequestBodyOperationEnum,
  BulkRequestBodyStatusEnum
} from '~/openapi-generator'
import { ResponseError } from '~/openapi-generator/runtime'
import { useCollectionBulkStatusData } from '~/loaders/collection-bulk-status'
import {
  buildBulkRequest,
  createDefaultBulkStatus,
  didBulkRunFail
} from './useCollectionBulk.helpers'

type SelectOption = {
  disabled?: boolean
  label: string
  value: string
}

const statusRefreshDelayMs = 1000

const statusOptions: SelectOption[] = [
  { label: 'Error', value: BulkRequestBodyStatusEnum.Error }
]

const operationOptions: SelectOption[] = [
  { label: 'Retry', value: BulkRequestBodyOperationEnum.Retry },
  { label: 'Cancel', value: BulkRequestBodyOperationEnum.Cancel, disabled: true },
  { label: 'Abandon', value: BulkRequestBodyOperationEnum.Abandon, disabled: true }
]

export function useCollectionBulk() {
  const enduroApi = useEnduroApi()
  const {
    data,
    isLoading: isLoadingStatus,
    reload
  } = useCollectionBulkStatusData()

  const selectedStatus = ref(BulkRequestBodyStatusEnum.Error)
  const selectedOperation = ref(BulkRequestBodyOperationEnum.Retry)
  const size = ref<number | null>(null)

  const isSubmitting = ref(false)
  const submitErrorMessage = ref('')
  const submitSuccessMessage = ref('')

  let statusTimer: number | null = null
  let isDisposed = false

  const status = computed(() => data.value?.status ?? createDefaultBulkStatus())
  const statusErrorMessage = computed(() => data.value?.errorMessage ?? '')

  const isRunning = computed(() => status.value.running)
  const hasCompletedRun = computed(() => Boolean(status.value.closedAt))
  const lastRunFailed = computed(() => didBulkRunFail(status.value.status))
  const canSubmit = computed(() => !isRunning.value && !isSubmitting.value)

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
      void reload()
    }, statusRefreshDelayMs)
  }

  async function loadStatus() {
    await reload()
  }

  function buildRequest(): BulkRequestBody {
    return buildBulkRequest({
      operation: selectedOperation.value,
      size: size.value,
      status: selectedStatus.value
    })
  }

  async function submit() {
    if (!canSubmit.value) return

    isSubmitting.value = true
    submitErrorMessage.value = ''
    submitSuccessMessage.value = ''

    try {
      await enduroApi.collections.bulk(buildRequest())
      submitSuccessMessage.value = 'Bulk operation submitted.'
      await reload()
    } catch (error) {
      if (error instanceof ResponseError && error.response.status === 409) {
        submitErrorMessage.value = 'A bulk operation started before this submission completed. Reloaded current status.'
        await reload()
      } else {
        submitErrorMessage.value = 'Could not submit the bulk operation.'
      }
    } finally {
      isSubmitting.value = false
    }
  }

  watch(isRunning, (running) => {
    if (running) {
      scheduleStatusRefresh()
    } else {
      clearStatusTimer()
    }
  }, { immediate: true })

  onBeforeUnmount(() => {
    isDisposed = true
    clearStatusTimer()
  })

  return {
    canSubmit,
    hasCompletedRun,
    isLoadingStatus,
    isRunning,
    isSubmitting,
    lastRunFailed,
    loadStatus,
    operationOptions,
    selectedOperation,
    selectedStatus,
    size,
    status,
    statusErrorMessage,
    statusOptions,
    submit,
    submitErrorMessage,
    submitSuccessMessage
  }
}
