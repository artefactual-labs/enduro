<script setup lang="ts">
const open = ref(false)
const monitor = useEnduroMonitor()
const versionLabel = useState<string>('enduroVersion', () => '')
const monitorVersionLabel = computed(() => versionLabel.value || '(version unavailable)')

const now = ref(Date.now())
let timer: number | null = null

onMounted(() => {
  timer = window.setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onBeforeUnmount(() => {
  if (!timer) return
  window.clearInterval(timer)
  timer = null
})

const shortDateFormatter = new Intl.DateTimeFormat(undefined, {
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit'
})

const longDateFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'medium'
})

function formatDate(value: string | null, long = true): string {
  if (!value) return 'N/A'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return 'N/A'
  return long ? longDateFormatter.format(parsed) : shortDateFormatter.format(parsed)
}

const retryInSeconds = computed(() => {
  if (!monitor.retryAt.value) return null
  const msLeft = new Date(monitor.retryAt.value).getTime() - now.value
  return Math.max(0, Math.ceil(msLeft / 1000))
})

type StatusMeta = {
  label: string
  badgeColor: 'success' | 'warning' | 'error'
  dotClass: string
  glowClass: string
}

const statusMeta = computed<StatusMeta>(() => {
  switch (monitor.status.value) {
    case 'connected':
      return {
        label: 'Connected',
        badgeColor: 'success',
        dotClass: 'bg-success-500',
        glowClass: 'bg-success-500/35'
      }
    case 'failed':
      return {
        label: 'Failed',
        badgeColor: 'error',
        dotClass: 'bg-error-500',
        glowClass: 'bg-error-500/35'
      }
    default:
      return {
        label: 'Connecting',
        badgeColor: 'warning',
        dotClass: 'bg-warning-500',
        glowClass: 'bg-warning-500/35'
      }
  }
})

const eventTimelineItems = computed(() => monitor.recentEvents.value.map(event => ({
  date: formatDate(event.receivedAt, false),
  title: event.type,
  description: [
    event.collectionId ? `Collection #${event.collectionId}` : 'Collection unknown',
    event.collectionStatus ? `status: ${event.collectionStatus}` : ''
  ].filter(Boolean).join(' · ')
})))

const formattedPingLatency = computed(() => {
  const value = monitor.pingLatencyMs.value
  if (value === null) return 'N/A'
  return `${Math.round(value)} ms`
})
</script>

<template>
  <UButton
    color="neutral"
    variant="ghost"
    size="sm"
    class="rounded-full border border-default/70 bg-elevated/70 px-2.5 hover:bg-elevated"
    @click="open = true"
  >
    <span class="inline-flex items-center gap-2">
      <span class="relative inline-flex size-2.5">
        <span
          class="absolute inset-0 rounded-full animate-ping"
          :class="statusMeta.glowClass"
        />
        <span
          class="relative inline-flex size-2.5 rounded-full border border-default/40"
          :class="statusMeta.dotClass"
        />
      </span>
      <span class="text-xs font-medium">{{ statusMeta.label }}</span>
    </span>
  </UButton>

  <USlideover
    v-model:open="open"
    title="Connection monitor"
    :description="`Live EventSource stream from ${monitor.endpoint.value}.`"
  >
    <template #body>
      <div class="space-y-4">
        <UCard>
          <div class="flex items-center justify-between gap-2">
            <p class="text-sm font-medium">
              SSE connection
            </p>
            <UBadge
              :label="statusMeta.label"
              :color="statusMeta.badgeColor"
              variant="subtle"
            />
          </div>

          <dl class="mt-4 grid grid-cols-3 gap-y-2 text-xs">
            <dt class="text-muted">
              Enduro version
            </dt>
            <dd class="col-span-2">
              <UBadge
                :label="monitorVersionLabel"
                color="neutral"
                variant="outline"
                class="font-mono"
              />
            </dd>

            <dt class="text-muted">
              Endpoint
            </dt>
            <dd class="col-span-2 break-all font-mono">
              {{ monitor.endpoint.value }}
            </dd>

            <dt class="text-muted">
              Connected since
            </dt>
            <dd class="col-span-2">
              {{ formatDate(monitor.connectedSince.value) }}
            </dd>

            <dt class="text-muted">
              Last event
            </dt>
            <dd class="col-span-2">
              {{ formatDate(monitor.lastEventAt.value) }}
            </dd>

            <dt class="text-muted">
              Total events
            </dt>
            <dd class="col-span-2">
              {{ monitor.totalEvents.value }}
            </dd>

            <dt class="text-muted">
              Est. latency
            </dt>
            <dd class="col-span-2">
              {{ formattedPingLatency }}
            </dd>

            <dt class="text-muted">
              Failures
            </dt>
            <dd class="col-span-2">
              {{ monitor.consecutiveFailures.value }}
            </dd>

            <dt class="text-muted">
              Next retry
            </dt>
            <dd class="col-span-2">
              <template v-if="monitor.retryAt.value">
                {{ formatDate(monitor.retryAt.value) }}
                <span
                  v-if="retryInSeconds !== null"
                  class="text-muted"
                >({{ retryInSeconds }}s)</span>
              </template>
              <template v-else>
                N/A
              </template>
            </dd>
          </dl>
        </UCard>

        <UCard>
          <template #header>
            <div class="flex items-center justify-between gap-2">
              <p class="text-sm font-medium">
                Recent monitor events (no ping)
              </p>
              <UBadge
                :label="String(eventTimelineItems.length)"
                color="neutral"
                variant="subtle"
              />
            </div>
          </template>

          <UTimeline
            v-if="eventTimelineItems.length"
            :items="eventTimelineItems"
          />
          <p
            v-else
            class="text-sm text-muted"
          >
            Waiting for events...
          </p>
        </UCard>
      </div>
    </template>

    <template #footer>
      <div class="flex w-full justify-between gap-2">
        <UButton
          label="Reconnect now"
          icon="i-lucide-refresh-cw"
          color="neutral"
          variant="outline"
          @click="monitor.reconnect"
        />
        <UButton
          label="Close"
          color="neutral"
          @click="open = false"
        />
      </div>
    </template>
  </USlideover>
</template>
