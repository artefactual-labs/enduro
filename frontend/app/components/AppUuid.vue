<script setup lang="ts">
import { copyTextToClipboard } from '~/utils/clipboard'

const props = withDefaults(defineProps<{
  emptyLabel?: string
  value?: string | null
}>(), {
  emptyLabel: 'N/A',
  value: null
})

const copied = ref(false)
const tooltipOpen = ref(false)
let copiedResetTimer: number | null = null

const displayValue = computed(() => props.value?.trim() || '')
const canCopy = computed(() => displayValue.value.length > 0)

async function onCopy() {
  if (!canCopy.value) return

  const success = await copyTextToClipboard(displayValue.value)
  if (!success) return

  copied.value = true
  tooltipOpen.value = true

  if (copiedResetTimer) {
    window.clearTimeout(copiedResetTimer)
  }

  copiedResetTimer = window.setTimeout(() => {
    copied.value = false
    tooltipOpen.value = false
    copiedResetTimer = null
  }, 1000)
}

function onTooltipOpenChange(value: boolean) {
  if (copied.value && !value) return
  tooltipOpen.value = value
}

onBeforeUnmount(() => {
  if (copiedResetTimer) {
    window.clearTimeout(copiedResetTimer)
  }
})
</script>

<template>
  <span class="inline-flex max-w-full min-w-0 items-start gap-2 align-top">
    <span
      v-if="canCopy"
      class="min-w-0 break-all font-mono text-sm text-toned"
    >
      {{ displayValue }}
    </span>
    <span
      v-else
      class="text-muted"
    >
      {{ emptyLabel }}
    </span>

    <UTooltip
      v-if="canCopy"
      :open="tooltipOpen"
      text="Copied!"
      :content="{ side: 'top' }"
      @update:open="onTooltipOpenChange"
    >
      <UButton
        :color="copied ? 'success' : 'neutral'"
        :icon="copied ? 'i-lucide-copy-check' : 'i-lucide-copy'"
        aria-label="Copy UUID"
        variant="ghost"
        size="xs"
        class="shrink-0"
        @click="onCopy"
      />
    </UTooltip>
  </span>
</template>
