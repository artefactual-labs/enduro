import type { EnduroStoredPipeline } from '~/openapi-generator'
import { usePipelinesListData } from '~/loaders/pipelines-list'

const refreshDelayMs = 5000

export function usePipelines() {
  const {
    data,
    error,
    isLoading,
    reload
  } = usePipelinesListData()

  const pipelines = computed<EnduroStoredPipeline[]>(() => data.value?.pipelines ?? [])
  const hasLoaded = computed(() => data.value !== undefined || error.value !== null)
  const errorMessage = computed(() => error.value?.message ?? '')

  let refreshTimer: number | null = null
  let isDisposed = false

  function clearRefreshTimer() {
    if (!refreshTimer) return
    window.clearTimeout(refreshTimer)
    refreshTimer = null
  }

  function scheduleRefresh() {
    if (!import.meta.client || isDisposed) return
    clearRefreshTimer()
    refreshTimer = window.setTimeout(() => {
      if (isDisposed) return
      refreshTimer = null
      void reload()
    }, refreshDelayMs)
  }

  async function loadPipelines() {
    await reload()
  }

  onMounted(() => {
    scheduleRefresh()
  })

  watch([data, error], () => {
    if (isDisposed) return
    scheduleRefresh()
  })

  onBeforeUnmount(() => {
    isDisposed = true
    clearRefreshTimer()
  })

  return {
    errorMessage,
    hasLoaded,
    isLoading,
    loadPipelines,
    pipelines
  }
}
