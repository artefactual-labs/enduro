import type { EnduroMonitorUpdate } from '~/openapi-generator'

export type EnduroConnectionStatus = 'connected' | 'connecting' | 'failed'

export type EnduroMonitorEvent = {
  receivedAt: string
  sourceTimestamp: string | null
  type: string
  collectionId: number | null
  collectionStatus: string | null
  raw: string
}

const FAILED_AFTER_RETRIES = 4
const MAX_RECENT_EVENTS = 100
const RETRY_DELAYS_MS = [1000, 2000, 5000, 10000, 15000, 30000]

let source: EventSource | null = null
let retryTimer: number | null = null

function nowIsoString(): string {
  return new Date().toISOString()
}

function trimTrailingSlash(value: string): string {
  if (!value || value === '/') return ''
  return value.replace(/\/+$/, '')
}

function normalizeMonitorUrl(rawUrl: string): string {
  if (!import.meta.client) return rawUrl

  try {
    const runtimeConfig = useRuntimeConfig()
    const appBase = trimTrailingSlash(String(runtimeConfig.app.baseURL ?? ''))
    const parsed = new URL(rawUrl, window.location.origin)

    parsed.pathname = parsed.pathname.replace(/\/+$/, '') || '/'

    if (appBase && parsed.origin === window.location.origin && parsed.pathname.startsWith(`${appBase}/`)) {
      parsed.pathname = parsed.pathname.slice(appBase.length) || '/'
    }

    if (parsed.origin === window.location.origin) {
      return `${parsed.pathname}${parsed.search}`
    }

    return parsed.toString()
  } catch {
    return rawUrl.replace(/\/+$/, '')
  }
}

function closeSource() {
  if (!source) return

  source.onopen = null
  source.onmessage = null
  source.onerror = null
  source.close()
  source = null
}

export function useEnduroMonitor() {
  const started = useState('enduro-monitor-started', () => false)
  const hasConnected = useState('enduro-monitor-has-connected', () => false)
  const status = useState<EnduroConnectionStatus>('enduro-monitor-status', () => 'connecting')
  const endpoint = useState('enduro-monitor-endpoint', () => '/collection/monitor')
  const connectedSince = useState<string | null>('enduro-monitor-connected-since', () => null)
  const lastEventAt = useState<string | null>('enduro-monitor-last-event-at', () => null)
  const retryAt = useState<string | null>('enduro-monitor-retry-at', () => null)
  const consecutiveFailures = useState('enduro-monitor-consecutive-failures', () => 0)
  const totalEvents = useState('enduro-monitor-total-events', () => 0)
  const pingLatencyMs = useState<number | null>('enduro-monitor-ping-latency-ms', () => null)
  const recentEvents = useState<EnduroMonitorEvent[]>('enduro-monitor-recent-events', () => [])

  function recordMessage(raw: string) {
    const receivedAt = nowIsoString()
    let sourceTimestamp: string | null = null
    let type = 'message'
    let collectionId: number | null = null
    let collectionStatus: string | null = null

    try {
      const payload = JSON.parse(raw) as Partial<EnduroMonitorUpdate> & {
        item?: { status?: string }
      }

      if (payload.timestamp instanceof Date) {
        sourceTimestamp = payload.timestamp.toISOString()
      } else if (typeof payload.timestamp === 'string') {
        sourceTimestamp = payload.timestamp
      }
      if (typeof payload.type === 'string') type = payload.type
      if (typeof payload.id === 'number') collectionId = payload.id
      if (typeof payload.item?.status === 'string') collectionStatus = payload.item.status
    } catch {
      type = 'invalid-json'
    }

    const nextEvent: EnduroMonitorEvent = {
      receivedAt,
      sourceTimestamp,
      type,
      collectionId,
      collectionStatus,
      raw
    }

    totalEvents.value += 1
    lastEventAt.value = receivedAt

    if (type === 'ping' && sourceTimestamp) {
      const sentAt = new Date(sourceTimestamp).getTime()
      if (!Number.isNaN(sentAt)) {
        pingLatencyMs.value = Math.max(0, Date.now() - sentAt)
      }
    } else {
      recentEvents.value = [nextEvent, ...recentEvents.value].slice(0, MAX_RECENT_EVENTS)
    }

    // Some SSE servers/proxies may delay the initial "open" transition until
    // the first data frame is received.
    if (!hasConnected.value) {
      hasConnected.value = true
      status.value = 'connected'
      connectedSince.value = receivedAt
      retryAt.value = null
      consecutiveFailures.value = 0
    }
  }

  function scheduleReconnect() {
    if (!import.meta.client || !started.value) return
    if (retryTimer) return

    closeSource()

    const failures = consecutiveFailures.value + 1
    const fallbackDelay = RETRY_DELAYS_MS[RETRY_DELAYS_MS.length - 1] ?? 30000
    const delay = RETRY_DELAYS_MS[Math.min(failures - 1, RETRY_DELAYS_MS.length - 1)] ?? fallbackDelay

    consecutiveFailures.value = failures
    retryAt.value = new Date(Date.now() + delay).toISOString()
    status.value = hasConnected.value && failures >= FAILED_AFTER_RETRIES ? 'failed' : 'connecting'

    retryTimer = window.setTimeout(() => {
      retryTimer = null
      connect()
    }, delay)
  }

  function connect() {
    if (!import.meta.client || !started.value) return

    closeSource()

    const { monitorUrl } = useEnduroApi()
    const resolvedMonitorUrl = normalizeMonitorUrl(monitorUrl)

    endpoint.value = resolvedMonitorUrl
    status.value = hasConnected.value && consecutiveFailures.value >= FAILED_AFTER_RETRIES
      ? 'failed'
      : 'connecting'

    const nextSource = new EventSource(resolvedMonitorUrl)
    source = nextSource

    nextSource.onopen = () => {
      if (source !== nextSource) return

      status.value = 'connected'
      hasConnected.value = true
      connectedSince.value = nowIsoString()
      retryAt.value = null
      consecutiveFailures.value = 0
    }

    nextSource.onmessage = (event: MessageEvent) => {
      if (source !== nextSource) return
      recordMessage(event.data)
    }

    nextSource.onerror = () => {
      if (source !== nextSource) return
      scheduleReconnect()
    }
  }

  function clearRetryTimer() {
    if (!retryTimer) return
    window.clearTimeout(retryTimer)
    retryTimer = null
  }

  function start() {
    if (!import.meta.client || started.value) return

    started.value = true
    clearRetryTimer()
    connect()
  }

  function reconnect() {
    if (!import.meta.client) return

    started.value = true
    clearRetryTimer()
    consecutiveFailures.value = 0
    retryAt.value = null
    status.value = 'connecting'
    connect()
  }

  return {
    status,
    endpoint,
    connectedSince,
    lastEventAt,
    retryAt,
    consecutiveFailures,
    totalEvents,
    pingLatencyMs,
    recentEvents,
    start,
    reconnect
  }
}
