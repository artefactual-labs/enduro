<template>

  <b-container>

    <!-- Alert shown when the API client failed. -->
    <template v-if="error">
      <b-alert show variant="warning" class="my-3">
        <h4 class="alert-heading">Search error</h4>
        We couldn't connect to the API server. You may want to try again in a few seconds.
        <hr />
        <b-button @click="retryButtonClicked" class="m-1">Retry</b-button>
      </b-alert>
    </template>

    <!-- Search form and results. -->
    <template v-else>

      <template v-if="results.length > 0">
        <table class="table table-bordered table-sm mt-4">
          <thead class="thead">
            <tr>
              <th scope="col">Name</th>
              <th>Capacity</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in results" v-bind:key="item.id">
              <th scope="row">{{ item.name }}</th>
              <td>{{ item.current }} / {{ item.capacity }}</td>
              <td>N / A</td>
            </tr>
          </tbody>
        </table>
      </template>

      <div v-if="results.length === 0">
        <b-alert show variant="info" class="my-3">
          <h4 class="alert-heading">No results</h4>
          We couldn’t find any collections matching your search criteria.
        </b-alert>
      </div>

    </template>

  </b-container>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { namespace } from 'vuex-class';
import * as PipelineStore from '../store/pipeline';

const pipelineStoreNs = namespace('pipeline');

@Component
export default class PipelineList extends Vue {

  @pipelineStoreNs.Getter(PipelineStore.GET_PIPELINE_ERROR)
  private error?: boolean;

  @pipelineStoreNs.Getter(PipelineStore.GET_SEARCH_RESULTS)
  private results: any;

  @pipelineStoreNs.Action(PipelineStore.SEARCH_PIPELINES)
  private search: any;

  private created() {
    this.search();
  }

  /**
   * Perform same search re-using all existing state.
   */
  private retryButtonClicked() {
    this.search();
  }

  /**
   * Forward user to the pipeline route.
   */
  private rowClicked(id: string) {
    this.$router.push({ name: 'pipeline', params: {id} });
  }
}
</script>

<style scoped lang="scss">

</style>

