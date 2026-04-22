<script setup lang="ts">
import type { ParsedWorkflowActivity } from '~/utils/workflow-history'

const props = defineProps<{
  activities: ParsedWorkflowActivity[]
  formatDateTime: (value: string | null) => string
  statusColor: (status: string) => 'success' | 'warning' | 'error' | 'neutral' | 'info'
}>()

function activityTimeLabel(activity: ParsedWorkflowActivity): string {
  if (activity.replayedAt) return 'Replayed'
  if (activity.startedAt) return 'Started'
  return 'Unavailable'
}

function activityTimeValue(activity: ParsedWorkflowActivity): string {
  if (activity.replayedAt) return props.formatDateTime(activity.replayedAt)
  if (activity.startedAt) return props.formatDateTime(activity.startedAt)
  return 'Time unavailable'
}

function activityDuration(activity: ParsedWorkflowActivity): string {
  return activity.durationSeconds ? `${activity.durationSeconds}s` : '—'
}

function activityAttempts(activity: ParsedWorkflowActivity): string {
  return activity.attempts > 1 ? String(activity.attempts) : '—'
}
</script>

<template>
  <div class="overflow-x-auto">
    <table class="min-w-full table-fixed">
      <colgroup>
        <col class="w-[5.5rem]">
        <col class="w-[18rem]">
        <col class="w-[8rem]">
        <col class="w-[13rem]">
        <col class="w-[7rem]">
        <col class="w-[7rem]">
        <col>
      </colgroup>
      <thead class="bg-elevated/30">
        <tr class="border-b border-default">
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            ID
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Activity
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Status
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Time
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Duration
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Attempts
          </th>
          <th
            scope="col"
            class="px-4 py-3 text-left text-sm font-semibold text-highlighted sm:px-5"
          >
            Details
          </th>
        </tr>
      </thead>
      <tbody class="divide-y divide-default">
        <tr
          v-for="activity in activities"
          :key="activity.id"
          class="app-table-row"
        >
          <td class="px-4 py-3 align-top text-sm sm:px-5">
            <UBadge
              :label="`#${activity.id}`"
              color="neutral"
              variant="outline"
              size="sm"
            />
          </td>
          <td class="px-4 py-3 align-top sm:px-5">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="break-words font-medium text-highlighted">
                  {{ activity.name }}
                </span>
                <UBadge
                  v-if="activity.isLocal"
                  label="Local"
                  color="neutral"
                  variant="outline"
                  size="sm"
                />
              </div>
            </div>
          </td>
          <td class="px-4 py-3 align-top sm:px-5">
            <UBadge
              v-if="activity.status"
              :label="activity.status.toUpperCase()"
              :color="statusColor(activity.status)"
              variant="subtle"
              size="sm"
            />
            <span
              v-else
              class="text-sm text-dimmed opacity-50"
            >—</span>
          </td>
          <td class="px-4 py-3 align-top text-sm sm:px-5">
            <div class="space-y-1">
              <div class="text-xs font-medium uppercase tracking-[0.14em] text-muted">
                {{ activityTimeLabel(activity) }}
              </div>
              <div class="text-highlighted">
                {{ activityTimeValue(activity) }}
              </div>
            </div>
          </td>
          <td class="px-4 py-3 align-top text-sm text-highlighted sm:px-5">
            <span :class="activity.durationSeconds ? 'text-highlighted' : 'text-dimmed opacity-50'">
              {{ activityDuration(activity) }}
            </span>
          </td>
          <td class="px-4 py-3 align-top text-sm text-highlighted sm:px-5">
            <span :class="activity.attempts > 1 ? 'text-highlighted' : 'text-dimmed opacity-50'">
              {{ activityAttempts(activity) }}
            </span>
          </td>
          <td class="px-4 py-3 align-top sm:px-5">
            <p
              v-if="activity.details"
              class="whitespace-pre-wrap break-words text-xs text-toned"
            >
              {{ activity.details }}
            </p>
            <span
              v-else
              class="text-sm text-dimmed opacity-50"
            >—</span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
