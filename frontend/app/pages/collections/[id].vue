<script lang="ts">
import { useCollectionPageData as collectionPageDataLoader } from '~/loaders/collection-details'

export const useCollectionPageData = collectionPageDataLoader
</script>

<script setup lang="ts">
const route = useRoute()
const { collectionName } = useCollectionDetails()
const backToCollectionsRoute = useCollectionsListLocation()

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
        />
      </UContainer>
    </div>

    <UContainer class="pt-6 pb-4">
      <NuxtPage />
    </UContainer>
  </div>
</template>
