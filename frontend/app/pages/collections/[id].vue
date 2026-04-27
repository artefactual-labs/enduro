<script lang="ts">
import { useCollectionPageData as collectionPageDataLoader } from '~/loaders/collection-details'

export const useCollectionPageData = collectionPageDataLoader
</script>

<script setup lang="ts">
const route = useRoute()
const { collectionName } = useCollectionDetails()
const backToCollectionsRoute = useCollectionsListLocation()
const autoReload = useCollectionWorkflowAutoReload()

const collectionId = computed(() => String(route.params.id ?? '0'))

const breadcrumbItems = computed(() => [
  {
    label: 'Collections',
    to: backToCollectionsRoute.value
  },
  {
    label: collectionName.value,
    to: `/collections/${collectionId.value}`
  }
])

const tabs = [
  {
    label: 'Overview',
    value: 'overview'
  },
  {
    label: 'Storage',
    value: 'storage'
  },
  {
    label: 'Workflow',
    value: 'workflow'
  }
]

const activeTab = computed({
  get() {
    if (route.path.endsWith('/storage')) return 'storage'
    return route.path.endsWith('/workflow') ? 'workflow' : 'overview'
  },
  set(value: string | number) {
    let next = `/collections/${collectionId.value}`
    if (value === 'storage') {
      next = `/collections/${collectionId.value}/storage`
    } else if (value === 'workflow') {
      next = `/collections/${collectionId.value}/workflow`
    }

    void navigateTo(next)
  }
})
</script>

<template>
  <div>
    <div class="bg-elevated/40 border-b border-default">
      <UContainer class="py-3">
        <UBreadcrumb :items="breadcrumbItems" />
      </UContainer>
    </div>

    <div>
      <UContainer class="pt-0 pb-0">
        <UTabs
          v-model="activeTab"
          :items="tabs"
          variant="link"
          :content="false"
          class="w-full"
          :ui="{
            list: 'px-0 sm:p-1',
            trigger: 'shrink-0 px-1.5 sm:px-3'
          }"
        >
          <template #list-trailing>
            <div
              v-if="activeTab === 'workflow'"
              class="ml-auto flex min-w-0 items-center gap-1.5"
            >
              <USwitch
                v-model="autoReload"
                aria-label="Auto-reload"
                color="primary"
                size="sm"
                class="shrink-0"
              />
              <span class="hidden min-w-0 truncate text-sm text-toned min-[400px]:inline">
                Auto-reload
              </span>
            </div>
          </template>
        </UTabs>
      </UContainer>
    </div>

    <UContainer class="pt-6 pb-4">
      <NuxtPage />
    </UContainer>
  </div>
</template>
