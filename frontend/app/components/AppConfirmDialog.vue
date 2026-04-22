<script setup lang="ts">
type ConfirmColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  description?: string
  confirmLabel?: string
  cancelLabel?: string
  confirmColor?: ConfirmColor
  pending?: boolean
}>(), {
  description: '',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  confirmColor: 'primary',
  pending: false
})

const emit = defineEmits<{
  'update:open': [value: boolean]
  'confirm': []
}>()

function updateOpen(value: boolean) {
  if (!value && props.pending) return
  emit('update:open', value)
}

function handleConfirm() {
  emit('confirm')
}
</script>

<template>
  <UModal
    :open="open"
    :title="title"
    :description="description"
    :close="!pending"
    :dismissible="!pending"
    class="max-w-md"
    @update:open="updateOpen"
  >
    <template #footer="{ close }">
      <div class="flex justify-end gap-2">
        <UButton
          :label="cancelLabel"
          color="neutral"
          variant="outline"
          :disabled="pending"
          @click="close()"
        />
        <UButton
          :label="confirmLabel"
          :color="confirmColor"
          :loading="pending"
          @click="handleConfirm"
        />
      </div>
    </template>
  </UModal>
</template>
