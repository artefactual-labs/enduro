<script setup lang="ts">
const { showConnectionMonitor } = useDisplaySettings()
const colorMode = useColorMode()
const appearanceOptions = [
  { label: 'System', value: 'system' },
  { label: 'Light', value: 'light' },
  { label: 'Dark', value: 'dark' }
]
</script>

<template>
  <USlideover
    title="Settings"
    description="Preferences are saved in this browser."
    :ui="{ content: 'max-w-sm' }"
  >
    <UButton
      icon="i-lucide-settings"
      aria-label="Open settings"
      color="neutral"
      variant="ghost"
    />

    <template #body>
      <div class="space-y-6">
        <URadioGroup
          v-model="colorMode.preference"
          legend="Appearance"
          aria-label="Appearance"
          :items="appearanceOptions"
          :disabled="colorMode.forced"
          orientation="horizontal"
          variant="table"
          indicator="hidden"
          :ui="{ legend: 'mb-2', item: 'flex-1 justify-center py-2' }"
        />
        <USeparator />
        <UFormField
          label="Show connection monitor"
          description="Show connection status in the footer."
          orientation="horizontal"
        >
          <USwitch v-model="showConnectionMonitor" />
        </UFormField>
      </div>
    </template>
  </USlideover>
</template>
