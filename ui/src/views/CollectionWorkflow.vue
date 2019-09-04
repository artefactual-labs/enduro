<template>
  <div class="collection-detail">
    <b-breadcrumb>
      <b-breadcrumb-item :to="{ name: 'collections' }">Collections</b-breadcrumb-item>
      <b-breadcrumb-item :to="{ name: 'collection', params: {id: this.$route.params.id} }">{{ name }}</b-breadcrumb-item>
      <b-breadcrumb-item>Workflow</b-breadcrumb-item>
    </b-breadcrumb>
    <div class="container-fluid">
      <a class="reload" href="#" v-on:click="reload()">Reload</a>
      <dl>
        <dt>Status</dt>
        <dd><b-badge>{{ history.status }}</b-badge></dd>
        <dt>History</dt>
        <dd>
          <table class="table table-bordered table-hover">
            <thead class="thead">
              <tr>
                <th scope="col">ID</th>
                <th scope="col">Type</th>
                <th scope="col">Timestamp</th>
                <th scope="col">Details</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in history.history.slice().reverse()" v-bind:key="item.id" @click="view(item)">
                <th scope="row">{{ item.id }}</th>
                <td>{{ item.type }}</td>
                <td>{{ item.details.timestamp | formatEpoch }}</td>
                <td>{{ renderDetails(item) }}</td>
              </tr>
            </tbody>
          </table>
        </dd>
      </dl>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Provide, Vue } from 'vue-property-decorator';
import { EnduroCollectionClient } from '../main';
import { CollectionShowResponseBody, CollectionWorkflowResponseBody, EnduroCollectionWorkflowHistoryResponseBody } from '../client/src';

@Component
export default class CollectionWorkflow extends Vue {

  private collection: any = {};
  private history: any = {history: []};
  private name: string = '';

  private mounted() {
    this.loadCollection();
    this.loadHistory();
  }

  private reload() {
    this.loadHistory();
  }

  private getName(body: CollectionShowResponseBody): string {
    if (body.name) {
      return body.name;
    }

    return body.id.toString();
  }

  private loadCollection() {
    EnduroCollectionClient.collectionShow({id: +this.$route.params.id}).then((response: CollectionShowResponseBody) => {
      this.collection = response;
      this.name = this.getName(response);
    });
  }

  private loadHistory() {
    return EnduroCollectionClient.collectionWorkflow({id: +this.$route.params.id}).then((response: CollectionWorkflowResponseBody) => {
      this.history = response;
    });
  }

  private renderDetails(event: EnduroCollectionWorkflowHistoryResponseBody): string {
    let ret = '';

    if (!event || !event.type) {
      return ret;
    }

    const attrs: any = event.details;
    if (event.type === 'ActivityTaskScheduled') {
      ret = 'Activity: ' + attrs.activityTaskScheduledEventAttributes.activityType.name;
    } else if (event.type === 'DecisionTaskScheduled') {
      const attempt: number = parseInt(attrs.decisionTaskScheduledEventAttributes.attempt, 10) + 1;
      ret = 'Attempts: ' + attempt;
    }

    return ret;
  }

}
</script>

<style scoped lang="scss">

.reload {
  float: right;
}

</style>
