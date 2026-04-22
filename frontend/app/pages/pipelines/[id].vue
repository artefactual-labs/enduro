<script lang="ts">
import { usePipelinePageData as pipelinePageDataLoader } from '~/loaders/pipeline-details'

export const usePipelinePageData = pipelinePageDataLoader
</script>

<script setup lang="ts">
const route = useRoute()
const {
  data,
  error,
  isLoading,
  reload
} = usePipelinePageData()

const routePipelineId = computed(() => String(route.params.id ?? ''))
const pipelineData = computed(() => data.value ?? null)
const pipeline = computed(() => pipelineData.value?.pipeline ?? null)
const pipelineId = computed(() => pipelineData.value?.pipelineId ?? routePipelineId.value)
const processingConfigurations = computed(() => pipelineData.value?.processingConfigurations ?? [])
const errorMessage = computed(() => error.value?.message ?? '')
const hasLoaded = computed(() => pipelineData.value !== null || error.value !== null)
const breadcrumbCurrentLabel = computed(() => {
  if (pipeline.value?.name) return pipeline.value.name
  if (!hasLoaded.value && isLoading.value) return 'Loading pipeline'
  if (errorMessage.value) return pipelineId.value || routePipelineId.value
  return 'Pipeline'
})

const breadcrumbItems = computed(() => [
  ...[
    {
      label: 'Pipelines',
      to: '/pipelines'
    }
  ],
  ...((!hasLoaded.value && !pipeline.value)
    ? []
    : [{
        label: breadcrumbCurrentLabel.value,
        to: `/pipelines/${pipelineId.value || routePipelineId.value}`
      }])
])

function statusColor(status: string | null | undefined) {
  return status === 'active' ? 'success' : 'warning'
}

const metadataItems = computed(() => [
  { key: 'identifier', label: 'Identifier', slot: 'identifier' },
  { key: 'capacity', label: 'Capacity', slot: 'capacity' }
])

useSeoMeta({
  title: () => pipeline.value?.name ? `Pipeline ${pipeline.value.name}` : `Pipeline ${pipelineId.value || 'unknown'}`
})
</script>

<template>
  <div>
    <div class="bg-elevated/40 border-b border-default">
      <UContainer class="py-3">
        <UBreadcrumb :items="breadcrumbItems" />
      </UContainer>
    </div>

    <UContainer class="py-4">
      <div class="space-y-4">
        <UAlert
          v-if="errorMessage"
          color="warning"
          variant="subtle"
          title="Pipeline unavailable"
          :description="errorMessage"
        >
          <template #actions>
            <UButton
              label="Retry"
              color="warning"
              variant="outline"
              size="sm"
              :loading="isLoading"
              @click="reload"
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
          v-else-if="pipeline"
          :ui="{ header: 'p-3 sm:px-5', body: 'p-0 sm:p-0' }"
        >
          <template #header>
            <div class="flex items-center justify-between gap-3">
              <div class="space-y-1">
                <p class="text-xs font-medium uppercase tracking-[0.2em] text-muted">
                  Pipeline
                </p>
                <h1 class="text-xl font-semibold text-highlighted">
                  {{ pipeline.name }}
                </h1>
              </div>

              <UBadge
                :label="(pipeline.status || 'unknown').toUpperCase()"
                :color="statusColor(pipeline.status)"
                variant="subtle"
              />
            </div>
          </template>

          <div>
            <AppMetadataList :items="metadataItems">
              <template #identifier>
                <AppUuid :value="pipeline.id || pipelineId" />
              </template>

              <template #capacity>
                <AppPipelineCapacity
                  :current="pipeline.current"
                  :capacity="pipeline.capacity"
                />
              </template>
            </AppMetadataList>

            <div class="border-t border-default">
              <AppMetadataList
                :items="[
                  { key: 'processingConfigurations', label: 'Processing configurations', slot: 'processingConfigurations' }
                ]"
              >
                <template #processingConfigurations>
                  <div class="space-y-3">
                    <p class="text-sm text-muted">
                      This list reflects the processing configurations reported by Archivematica for this pipeline.
                    </p>

                    <div
                      v-if="processingConfigurations.length"
                      class="flex flex-wrap gap-2"
                    >
                      <UBadge
                        v-for="item in processingConfigurations"
                        :key="item"
                        :label="item"
                        color="neutral"
                        variant="outline"
                      />
                    </div>
                    <p
                      v-else
                      class="text-sm text-muted"
                    >
                      Archivematica did not report any processing configurations for this pipeline.
                    </p>
                  </div>
                </template>
              </AppMetadataList>
            </div>
          </div>
        </UCard>
      </div>
    </UContainer>
  </div>
</template>
