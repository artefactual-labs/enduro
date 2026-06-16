<script setup lang="ts">
import type { BatchBrowseEntry } from '~/openapi-generator'

type DirectoryTreeItem = {
  absolutePath: string
  children?: DirectoryTreeItem[]
  disabled?: boolean
  icon?: string
  label: string
  loaded: boolean
  loading: boolean
  path: string
  truncated: boolean
}

const props = defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'select': [value: string]
}>()

const enduroApi = useEnduroApi()

const errorMessage = ref('')
const expandedKeys = ref<string[]>([])
const selectedItem = ref<DirectoryTreeItem | undefined>()

const rootItem = ref<DirectoryTreeItem>({
  absolutePath: '',
  children: [createLoadingItem()],
  icon: 'i-lucide-folder',
  label: '',
  loaded: false,
  loading: false,
  path: '',
  truncated: false
})

const items = computed(() => rootItem.value.children ?? [])
const selectedAbsolutePath = computed(() => selectedItem.value?.absolutePath ?? '')
const hasTruncatedDirectory = computed(() => findTruncatedItem(rootItem.value) !== undefined)
const modalUi = {
  body: 'flex min-h-0 flex-1 flex-col',
  footer: 'shrink-0',
  header: 'shrink-0'
}
const treeUi = {
  link: 'data-[selected]:z-10 data-[selected]:rounded-md data-[selected]:ring-2 data-[selected]:ring-amber-700/80 data-[selected]:before:bg-amber-50/80 dark:data-[selected]:before:bg-amber-950/30'
}

watch(() => props.open, (open) => {
  if (open) {
    selectedItem.value = undefined
    void loadDirectory(rootItem.value)
  }
}, { immediate: true })

function updateOpen(value: boolean) {
  emit('update:open', value)
}

function getItemKey(item: DirectoryTreeItem): string {
  return item.path || '.'
}

function createLoadingItem(): DirectoryTreeItem {
  return {
    absolutePath: '',
    disabled: true,
    icon: 'i-lucide-folder',
    label: 'Loading directories...',
    loaded: true,
    loading: false,
    path: '__loading__',
    truncated: false
  }
}

function createEmptyItem(parentPath: string): DirectoryTreeItem {
  return {
    absolutePath: '',
    disabled: true,
    icon: 'i-lucide-folder-minus',
    label: 'No child directories',
    loaded: true,
    loading: false,
    path: `${parentPath || '.'}/__empty__`,
    truncated: false
  }
}

function createDirectoryItem(entry: BatchBrowseEntry): DirectoryTreeItem {
  return {
    absolutePath: entry.absolutePath,
    children: [createLoadingItem()],
    icon: 'i-lucide-folder',
    label: entry.name,
    loaded: false,
    loading: false,
    path: entry.path,
    truncated: false
  }
}

async function loadDirectory(item: DirectoryTreeItem) {
  if (item.disabled || item.loaded || item.loading) return

  item.loading = true
  errorMessage.value = ''

  try {
    const result = await enduroApi.batches.browse(item.path ? { path: item.path } : {})
    item.absolutePath = result.absolutePath
    item.path = result.path
    item.loaded = true
    item.truncated = result.truncated
    item.children = result.entries.length > 0
      ? result.entries.map(createDirectoryItem)
      : [createEmptyItem(result.path)]
  } catch {
    errorMessage.value = 'Could not load directories.'
  } finally {
    item.loading = false
  }
}

function updateExpandedKeys(value: string[]) {
  expandedKeys.value = value

  for (const key of value) {
    const item = findItemByKey(rootItem.value, key)
    if (item) {
      void loadDirectory(item)
    }
  }
}

function updateSelectedItem(item: DirectoryTreeItem | undefined) {
  if (!item || item.disabled) return
  selectedItem.value = item
}

function useSelectedDirectory() {
  if (!selectedAbsolutePath.value) return
  emit('select', selectedAbsolutePath.value)
  emit('update:open', false)
}

function findItemByKey(item: DirectoryTreeItem, key: string): DirectoryTreeItem | undefined {
  if (getItemKey(item) === key) return item

  for (const child of item.children ?? []) {
    const found = findItemByKey(child, key)
    if (found) return found
  }

  return undefined
}

function findTruncatedItem(item: DirectoryTreeItem): DirectoryTreeItem | undefined {
  if (item.truncated) return item

  for (const child of item.children ?? []) {
    const found = findTruncatedItem(child)
    if (found) return found
  }

  return undefined
}
</script>

<template>
  <UModal
    :open="open"
    title="Browse source directory"
    description="Choose a source directory for this batch import."
    class="h-[calc(100dvh-2rem)] max-w-3xl sm:h-[calc(100dvh-4rem)]"
    :ui="modalUi"
    @update:open="updateOpen"
  >
    <template #body>
      <div class="flex min-h-0 flex-1 flex-col gap-4">
        <UAlert
          v-if="errorMessage"
          color="warning"
          variant="subtle"
          :description="errorMessage"
        />

        <UAlert
          v-if="hasTruncatedDirectory"
          color="warning"
          variant="subtle"
          description="This directory has more than 1000 child directories. Only the first 1000 are shown."
        />

        <div class="min-h-0 flex-1 overflow-auto rounded-md border border-default p-2">
          <UTree
            :items="items"
            :model-value="selectedItem"
            :expanded="expandedKeys"
            :get-key="getItemKey"
            :ui="treeUi"
            selection-behavior="replace"
            @update:model-value="updateSelectedItem"
            @update:expanded="updateExpandedKeys"
          />
        </div>

        <div
          v-if="selectedAbsolutePath"
          class="space-y-2"
        >
          <p class="text-xs font-medium uppercase tracking-[0.14em] text-muted">
            Selected path
          </p>
          <pre class="min-h-9 overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-elevated px-3 py-2 font-mono text-xs leading-relaxed text-highlighted">{{ selectedAbsolutePath }}</pre>
        </div>
        <p
          v-else
          class="text-sm text-muted"
        >
          No directory selected.
        </p>
      </div>
    </template>

    <template #footer="{ close }">
      <div class="flex justify-end gap-2">
        <UButton
          label="Cancel"
          color="neutral"
          variant="outline"
          @click="close()"
        />
        <UButton
          label="Use directory"
          color="primary"
          :disabled="!selectedAbsolutePath"
          @click="useSelectedDirectory"
        />
      </div>
    </template>
  </UModal>
</template>
