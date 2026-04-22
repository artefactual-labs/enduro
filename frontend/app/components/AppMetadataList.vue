<script setup lang="ts">
type MetadataItem = {
  key: string
  label: string
  slot?: string
  value?: string | null
  valueClass?: string
}

const props = defineProps<{
  items: MetadataItem[]
  layout?: 'split' | 'stacked'
  labelWidth?: string
}>()

const visibleItems = computed(() => props.items.filter(item => item.value !== undefined || item.slot))
const rowClass = computed(() => {
  if (props.layout === 'stacked') {
    return 'space-y-1 px-4 py-3 sm:px-5'
  }

  return 'grid gap-1 px-4 py-3 sm:px-5 md:grid-cols-[var(--app-metadata-label-width,_12rem)_minmax(0,1fr)] md:gap-6'
})
const rowStyle = computed(() => props.labelWidth ? { '--app-metadata-label-width': props.labelWidth } : undefined)
</script>

<template>
  <dl class="divide-y divide-default">
    <div
      v-for="item in visibleItems"
      :key="item.key"
      :class="rowClass"
      :style="rowStyle"
    >
      <dt
        class="text-sm text-muted"
        :class="layout === 'stacked' ? 'text-xs font-medium uppercase tracking-[0.14em]' : ''"
      >
        {{ item.label }}
      </dt>
      <dd
        class="min-w-0 text-sm text-highlighted break-words"
        :class="[item.valueClass, layout === 'stacked' ? 'text-base' : '']"
      >
        <slot :name="item.slot || `${item.key}-value`">
          {{ item.value }}
        </slot>
      </dd>
    </div>
  </dl>
</template>
