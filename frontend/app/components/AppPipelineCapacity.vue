<script setup lang="ts">
import { summarizePipelineCapacity } from '~/utils/pipeline-capacity'

const props = defineProps<{
  capacity?: number | null
  current?: number | null
}>()

const summary = computed(() => summarizePipelineCapacity(props.current, props.capacity))
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-end gap-2">
      <span class="font-mono text-base font-medium leading-none text-highlighted">
        {{ summary.currentClamped }} / {{ summary.capacity }}
      </span>
      <span class="text-[11px] font-medium uppercase tracking-[0.18em] text-muted">
        slots
      </span>
    </div>

    <UProgress
      :model-value="summary.currentClamped"
      :max="summary.capacity"
      :color="summary.color"
      size="lg"
    />
  </div>
</template>
