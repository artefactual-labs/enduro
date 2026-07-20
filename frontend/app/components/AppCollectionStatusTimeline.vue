<script setup lang="ts">
import type { EnduroCollectionStatusHistory } from '~/openapi-generator'
import {
  buildCollectionStatusPeriods,
  collectionStatusPeriodHasDuration,
  formatCollectionStatusDuration,
  summarizeCollectionStatusPeriods
} from '~/utils/collection-status-history'
import {
  collectionStatusPresentation,
  collectionStatusReasonLabel
} from '~/utils/collection-status'

const props = defineProps<{
  history: EnduroCollectionStatusHistory
  runId?: string | null
}>()

const now = ref(new Date())
let clock: number | null = null

const periods = computed(() => buildCollectionStatusPeriods(
  props.history.transitions,
  props.runId,
  now.value
))
const totals = computed(() => summarizeCollectionStatusPeriods(periods.value))
const isUnavailable = computed(() => props.history.availability === 'unavailable')
const isPartial = computed(() => props.history.availability === 'partial')
const transitionHistoryOpen = ref(false)

const dateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short'
})

function formatDateTime(value: Date | null): string {
  if (!value || Number.isNaN(value.getTime())) return 'Time unavailable'
  return dateTimeFormatter.format(value)
}

function indicatorClass(status: string): string {
  switch (collectionStatusPresentation(status).color) {
    case 'success': return 'border-success bg-success/15 text-success'
    case 'error': return 'border-error bg-error/15 text-error'
    case 'warning': return 'border-warning bg-warning/15 text-warning'
    case 'info': return 'border-info bg-info/15 text-info'
    default: return 'border-default bg-elevated text-muted'
  }
}

onMounted(() => {
  clock = window.setInterval(() => {
    now.value = new Date()
  }, 30_000)
})

onBeforeUnmount(() => {
  if (clock !== null) window.clearInterval(clock)
})
</script>

<template>
  <UAlert
    v-if="isUnavailable"
    color="neutral"
    variant="subtle"
    icon="i-lucide-history"
    title="Status history is not available for this workflow run"
    description="Enduro started recording transitions after this workflow was started. Complete history will be available for collections started or retried after the upgrade."
  />

  <div
    v-else-if="periods.length"
    class="space-y-5"
  >
    <UAlert
      v-if="isPartial"
      color="warning"
      variant="subtle"
      icon="i-lucide-history"
      title="Status history is incomplete"
      description="Only transitions recorded after the Enduro upgrade are shown. Earlier statuses and their durations are not included."
    />

    <div class="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(14rem,1fr)]">
      <ol
        aria-label="Collection status timeline"
        class="space-y-0"
      >
        <li
          v-for="(period, index) in periods"
          :key="period.id"
          class="relative grid grid-cols-[2.25rem_minmax(0,1fr)] gap-3 pb-6 last:pb-0"
        >
          <div
            v-if="index < periods.length - 1"
            aria-hidden="true"
            class="absolute bottom-0 left-[1.0625rem] top-9 w-px bg-default"
          />
          <div
            :class="indicatorClass(period.status)"
            class="relative z-[1] flex size-9 items-center justify-center rounded-full border"
          >
            <UIcon
              :name="collectionStatusPresentation(period.status).icon"
              class="size-4"
            />
          </div>

          <div class="min-w-0 pt-1">
            <div class="flex flex-wrap items-center gap-2">
              <UBadge
                :label="collectionStatusPresentation(period.status).label"
                :color="collectionStatusPresentation(period.status).color"
                variant="subtle"
              />
              <UBadge
                v-if="period.isCurrent"
                label="Current"
                color="neutral"
                variant="outline"
                size="sm"
              />
              <span
                v-if="collectionStatusPeriodHasDuration(period)"
                class="text-sm font-medium text-highlighted"
              >
                for {{ formatCollectionStatusDuration(period.durationMs) }}
              </span>
            </div>
            <p class="mt-1 text-sm text-muted">
              Entered {{ formatDateTime(period.startedAt) }}
              <template v-if="period.endedAt">
                · left {{ formatDateTime(period.endedAt) }}
              </template>
            </p>
            <p class="mt-1 text-sm text-toned">
              {{ collectionStatusReasonLabel(period.reason) }}
            </p>
          </div>
        </li>
      </ol>

      <div>
        <h4 class="text-sm font-semibold text-highlighted">
          {{ isPartial ? 'Recorded time by status' : 'Time by status' }}
        </h4>
        <dl class="mt-2 divide-y divide-default rounded-lg border border-default">
          <div
            v-for="total in totals"
            :key="total.status"
            class="flex items-center justify-between gap-4 px-3 py-2.5"
          >
            <dt class="flex items-center gap-2 text-sm text-toned">
              <UIcon
                :name="collectionStatusPresentation(total.status).icon"
                class="size-4"
              />
              {{ collectionStatusPresentation(total.status).label }}
            </dt>
            <dd class="text-sm font-medium text-highlighted">
              {{ formatCollectionStatusDuration(total.durationMs) }}
            </dd>
          </div>
        </dl>
      </div>
    </div>

    <UCollapsible v-model:open="transitionHistoryOpen">
      <UButton
        :label="transitionHistoryOpen ? 'Hide exact transition history' : 'Show exact transition history'"
        icon="i-lucide-list"
        :trailing-icon="transitionHistoryOpen ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
        color="neutral"
        :variant="transitionHistoryOpen ? 'soft' : 'outline'"
        size="sm"
        :aria-pressed="transitionHistoryOpen"
      />

      <template #content>
        <div class="mt-3 overflow-x-auto rounded-lg border border-default">
          <table class="min-w-full">
            <thead class="bg-elevated/30">
              <tr class="border-b border-default">
                <th
                  class="px-3 py-2 text-left text-xs font-semibold text-highlighted"
                  scope="col"
                >
                  Status
                </th>
                <th
                  class="px-3 py-2 text-left text-xs font-semibold text-highlighted"
                  scope="col"
                >
                  Entered
                </th>
                <th
                  class="px-3 py-2 text-left text-xs font-semibold text-highlighted"
                  scope="col"
                >
                  Left
                </th>
                <th
                  class="px-3 py-2 text-left text-xs font-semibold text-highlighted"
                  scope="col"
                >
                  Time in status
                </th>
                <th
                  class="px-3 py-2 text-left text-xs font-semibold text-highlighted"
                  scope="col"
                >
                  Reason
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-default">
              <tr
                v-for="period in periods"
                :key="`table-${period.id}`"
                class="app-table-row"
              >
                <td class="px-3 py-2 text-sm">
                  {{ collectionStatusPresentation(period.status).label }}
                </td>
                <td class="px-3 py-2 text-sm text-muted whitespace-nowrap">
                  {{ formatDateTime(period.startedAt) }}
                </td>
                <td class="px-3 py-2 text-sm text-muted whitespace-nowrap">
                  {{ period.endedAt ? formatDateTime(period.endedAt) : 'Current' }}
                </td>
                <td class="px-3 py-2 text-sm font-medium whitespace-nowrap">
                  {{ collectionStatusPeriodHasDuration(period) ? formatCollectionStatusDuration(period.durationMs) : '—' }}
                </td>
                <td class="px-3 py-2 text-sm text-toned">
                  {{ collectionStatusReasonLabel(period.reason) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </UCollapsible>
  </div>

  <UAlert
    v-else
    color="neutral"
    variant="subtle"
    title="No status transitions were returned"
    description="This workflow run is marked as recorded, but its transition history is empty."
  />
</template>
