import type { TableRow } from '@nuxt/ui'
import type { EnduroStoredCollection } from '~/openapi-generator'
import { useCollectionsListData } from '~/loaders/collections-list'
import {
  dateOptions,
  fieldOptions,
  isValidCollectionSearchQuery,
  normalizeDateFilter,
  normalizeFieldFilter,
  normalizeStatusFilter,
  statusOptions,
  type DateFilter,
  type FieldFilter,
  type StatusFilter
} from '~/loaders/collections-list.helpers'

type CollectionStatus = EnduroStoredCollection['status']
type CollectionsRouteQuery = Record<string, string>

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

function readQueryValue(value: unknown): string {
  if (typeof value === 'string') return value
  if (Array.isArray(value) && typeof value[0] === 'string') return value[0]
  return ''
}

function routeQuerySnapshot(queryLike: Record<string, unknown>): string {
  const entries: Array<{ key: string, value: string }> = []

  for (const [key, value] of Object.entries(queryLike)) {
    const normalizedValue = readQueryValue(value)
    if (normalizedValue === '') continue
    entries.push({ key, value: normalizedValue })
  }

  entries.sort((left, right) => left.key.localeCompare(right.key))
  return entries.map(entry => `${entry.key}=${entry.value}`).join('&')
}

export function useCollectionsListLocation() {
  const routeQuery = useState<CollectionsRouteQuery>('collections-browser-route-query', () => ({}))

  return computed(() => ({
    path: '/collections',
    query: { ...routeQuery.value }
  }))
}

export function useCollectionsBrowser() {
  const monitor = useEnduroMonitor()
  const route = useRoute()
  const router = useRouter()
  const {
    data,
    error,
    isLoading,
    reload
  } = useCollectionsListData()

  const selectedStatus = ref<StatusFilter>('all')
  const selectedDate = ref<DateFilter>('all')
  const selectedField = ref<FieldFilter>('name')
  const query = ref('')
  const validQuery = ref<boolean | null>(null)

  const routeQuery = useState<CollectionsRouteQuery>('collections-browser-route-query', () => ({}))
  const seenCursors = useState<string[]>('collections-browser-seen-cursors', () => [])

  const activeFilterSignature = ref('')

  let sseRefreshTimer: number | null = null
  let suppressRouteSync = false

  const queryHelp = computed(() => {
    switch (selectedField.value) {
      case 'name':
        return 'Prefix and case-insensitive name matching, e.g. "DPJ-SIP-97".'
      case 'original_id':
        return 'Exact matching.'
      default:
        return 'Exact matching, use UUID identifiers.'
    }
  })

  const queryError = computed(() => {
    if (validQuery.value !== false) return ''
    return 'Invalid UUID format for this field.'
  })

  const rows = computed(() => data.value?.items ?? [])
  const hasRows = computed(() => rows.value.length > 0)
  const hasError = computed(() => Boolean(error.value))
  const currentCursor = computed(() => readQueryValue(route.query.cursor))
  const nextCursor = computed(() => data.value?.nextCursor ?? null)
  const canGoPrev = computed(() => seenCursors.value.length > 0)
  const canGoNext = computed(() => Boolean(nextCursor.value))
  const showPager = computed(() => canGoPrev.value || canGoNext.value)

  function statusColor(status: CollectionStatus): 'success' | 'warning' | 'error' | 'neutral' | 'info' {
    if (status === 'done') return 'success'
    if (status === 'error') return 'error'
    if (status === 'in progress') return 'warning'
    if (status === 'pending' || status === 'queued') return 'info'
    return 'neutral'
  }

  function formatDateTime(value: Date | string | null | undefined): string {
    if (!value) return 'N/A'
    const date = value instanceof Date ? value : new Date(value)
    if (Number.isNaN(date.getTime())) return 'N/A'
    return dateTimeFormatter.format(date)
  }

  function buildFilterRouteQuery(): CollectionsRouteQuery {
    const next: CollectionsRouteQuery = {}
    if (selectedStatus.value !== 'all') next.status = selectedStatus.value
    if (selectedDate.value !== 'all') next.date = selectedDate.value
    if (selectedField.value !== 'name') next.field = selectedField.value

    const trimmedQuery = query.value.trim()
    if (trimmedQuery) next.q = trimmedQuery

    return next
  }

  function saveRouteQuery(queryLike: CollectionsRouteQuery) {
    routeQuery.value = { ...queryLike }
  }

  function validateQuery(field: FieldFilter, value: string): boolean {
    return isValidCollectionSearchQuery(field, value)
  }

  function applyRouteFilters(queryLike: Record<string, unknown>) {
    suppressRouteSync = true

    const status = readQueryValue(queryLike.status)
    const date = readQueryValue(queryLike.date)
    const field = readQueryValue(queryLike.field)
    const search = readQueryValue(queryLike.q)

    selectedStatus.value = normalizeStatusFilter(status)
    selectedDate.value = normalizeDateFilter(date)
    selectedField.value = normalizeFieldFilter(field)
    query.value = search
    validQuery.value = validateQuery(selectedField.value, search.trim()) ? null : false

    suppressRouteSync = false
  }

  async function applyRouteQuery(
    nextFilterQuery: CollectionsRouteQuery,
    routeOptions: {
      cursor?: string
      forceReload?: boolean
      scrollToTop?: boolean
    } = {}
  ) {
    const nextRouteQuery: CollectionsRouteQuery = { ...nextFilterQuery }
    if (routeOptions.cursor) nextRouteQuery.cursor = routeOptions.cursor

    saveRouteQuery(nextFilterQuery)

    const currentSignature = routeQuerySnapshot(route.query as unknown as Record<string, unknown>)
    const nextSignature = routeQuerySnapshot(nextRouteQuery)

    if (currentSignature === nextSignature) {
      if (routeOptions.forceReload) {
        await reload()
      }
    } else {
      await router.replace({ query: nextRouteQuery })
    }

    if (routeOptions.scrollToTop && import.meta.client) {
      window.scrollTo({ top: 0, left: 0, behavior: 'auto' })
    }
  }

  function resetCursors() {
    seenCursors.value = []
  }

  function scheduleSseRefresh() {
    if (sseRefreshTimer) {
      window.clearTimeout(sseRefreshTimer)
    }

    sseRefreshTimer = window.setTimeout(() => {
      sseRefreshTimer = null
      void reload()
    }, 1000)
  }

  function onSubmit() {
    const trimmedQuery = query.value.trim()
    const isValid = validateQuery(selectedField.value, trimmedQuery)
    validQuery.value = isValid ? null : false
    if (!isValid) return

    resetCursors()
    void applyRouteQuery(buildFilterRouteQuery(), {
      forceReload: true,
      scrollToTop: true
    })
  }

  function onReset() {
    suppressRouteSync = true
    selectedStatus.value = 'all'
    selectedDate.value = 'all'
    selectedField.value = 'name'
    query.value = ''
    validQuery.value = null
    suppressRouteSync = false

    resetCursors()
    void applyRouteQuery({}, {
      forceReload: true,
      scrollToTop: true
    })
  }

  function onRetry() {
    void reload()
  }

  function onGoHome() {
    if (!canGoPrev.value) return
    resetCursors()
    void applyRouteQuery(buildFilterRouteQuery(), { scrollToTop: true })
  }

  function onGoPrev() {
    if (!canGoPrev.value) return

    const previousCursor = seenCursors.value.pop() ?? ''
    void applyRouteQuery(buildFilterRouteQuery(), {
      cursor: previousCursor,
      scrollToTop: true
    })
  }

  function onGoNext() {
    if (!nextCursor.value) return

    seenCursors.value.push(currentCursor.value)
    void applyRouteQuery(buildFilterRouteQuery(), {
      cursor: nextCursor.value,
      scrollToTop: true
    })
  }

  function onCollectionRowSelect(_event: Event, row: TableRow<EnduroStoredCollection>) {
    void navigateTo(`/collections/${row.original.id}`)
  }

  watch(selectedStatus, () => {
    if (suppressRouteSync) return
    resetCursors()
    void applyRouteQuery(buildFilterRouteQuery())
  })

  watch(selectedDate, () => {
    if (suppressRouteSync) return
    resetCursors()
    void applyRouteQuery(buildFilterRouteQuery())
  })

  watch(selectedField, () => {
    if (suppressRouteSync) return
    validQuery.value = null
  })

  watch(query, () => {
    if (suppressRouteSync) return
    validQuery.value = null
  })

  watch(() => route.query, () => {
    const nextFilterQuery = {
      status: readQueryValue(route.query.status),
      date: readQueryValue(route.query.date),
      field: readQueryValue(route.query.field),
      q: readQueryValue(route.query.q)
    }
    const nextFilterSignature = routeQuerySnapshot(nextFilterQuery)

    applyRouteFilters(route.query as unknown as Record<string, unknown>)
    saveRouteQuery(buildFilterRouteQuery())

    if (activeFilterSignature.value && activeFilterSignature.value !== nextFilterSignature) {
      resetCursors()
    }

    activeFilterSignature.value = nextFilterSignature
  }, { immediate: true })

  watch(() => monitor.recentEvents.value[0]?.receivedAt, () => {
    const latest = monitor.recentEvents.value[0]
    if (!latest) return
    if (latest.type === 'collection:created' || latest.type === 'collection:updated') {
      scheduleSseRefresh()
    }
  })

  onBeforeUnmount(() => {
    if (!sseRefreshTimer) return
    window.clearTimeout(sseRefreshTimer)
    sseRefreshTimer = null
  })

  return {
    statusOptions,
    dateOptions,
    fieldOptions,
    selectedStatus,
    selectedDate,
    selectedField,
    query,
    isLoading,
    hasError,
    validQuery,
    rows,
    queryHelp,
    queryError,
    hasRows,
    canGoPrev,
    canGoNext,
    showPager,
    statusColor,
    formatDateTime,
    onSubmit,
    onReset,
    onRetry,
    onGoHome,
    onGoPrev,
    onGoNext,
    onCollectionRowSelect
  }
}
