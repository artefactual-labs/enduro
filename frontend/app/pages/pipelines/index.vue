<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { EnduroStoredPipeline } from '~/openapi-generator'

const {
  errorMessage,
  hasLoaded,
  isLoading,
  loadPipelines,
  pipelines
} = usePipelines()

const columns: TableColumn<EnduroStoredPipeline>[] = [
  {
    accessorKey: 'name',
    header: 'Name'
  },
  {
    id: 'capacity',
    header: 'Capacity'
  },
  {
    accessorKey: 'status',
    header: 'Status'
  }
]

function statusColor(status: string | null | undefined) {
  return status === 'active' ? 'success' : 'warning'
}

function onPipelineRowSelect(_event: Event, row: { original: EnduroStoredPipeline }) {
  if (!row.original.id) return
  void navigateTo(`/pipelines/${row.original.id}`)
}
</script>

<script lang="ts">
export { usePipelinesListData } from '~/loaders/pipelines-list'
</script>

<template>
  <AppPageContainer>
    <div>
      <h1 class="text-xl font-semibold text-highlighted">
        Pipelines
      </h1>
    </div>

    <UAlert
      v-if="errorMessage"
      color="warning"
      variant="subtle"
      title="Pipeline status unavailable"
      :description="errorMessage"
    >
      <template #actions>
        <UButton
          label="Retry"
          color="warning"
          variant="outline"
          size="sm"
          :loading="isLoading"
          @click="loadPipelines"
        />
      </template>
    </UAlert>

    <UCard v-else-if="!hasLoaded && isLoading">
      <div class="space-y-3">
        <USkeleton class="h-5 w-32" />
        <USkeleton class="h-4 w-full" />
        <USkeleton class="h-4 w-5/6" />
      </div>
    </UCard>

    <UCard
      v-else-if="pipelines.length"
      :ui="{ body: 'p-0 sm:p-0' }"
    >
      <UTable
        :data="pipelines"
        :columns="columns"
        :meta="{
          class: {
            tr: 'app-table-row cursor-pointer'
          }
        }"
        :on-select="onPipelineRowSelect"
        :ui="{
          th: 'px-4 py-3 text-sm text-highlighted',
          td: 'px-4 py-3 text-sm text-muted whitespace-nowrap'
        }"
      >
        <template #name-cell="{ row }">
          <UButton
            :label="row.original.name"
            :to="row.original.id ? `/pipelines/${row.original.id}` : undefined"
            color="primary"
            variant="link"
            size="xs"
            class="p-0 leading-none"
          />
        </template>

        <template #capacity-cell="{ row }">
          {{ row.original.current ?? 0 }} / {{ row.original.capacity ?? 0 }}
        </template>

        <template #status-cell="{ row }">
          <UBadge
            :label="(row.original.status || 'unknown').toUpperCase()"
            :color="statusColor(row.original.status)"
            variant="subtle"
          />
        </template>
      </UTable>
    </UCard>

    <UAlert
      v-else
      color="info"
      variant="subtle"
      title="No pipelines"
      description="No pipelines are currently available."
    />
  </AppPageContainer>
</template>
