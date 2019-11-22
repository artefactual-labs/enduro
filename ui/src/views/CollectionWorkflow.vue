<template>
  <b-container>
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
          <dt>Activity summary</dt>
          <dd>
            <table class="table table-bordered table-hover table-sm">
              <thead class="thead">
                <tr>
                  <th scope="col">Name</th>
                  <th scope="col">Started</th>
                  <th scope="col">Duration (seconds)</th>
                  <th scope="col">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(item, index) in activities" v-bind:key="index">
                  <td scope="row">{{ item.name }}</td>
                  <td>{{ item.started | formatEpoch }}</td>
                  <td>{{ item.duration }}</td>
                  <td><en-collection-status-badge :status="item.status"/></td>
                </tr>
              </tbody>
            </table>
          </dd>
          <dt>History</dt>
          <dd>
            <table class="table table-bordered table-hover table-sm">
              <thead class="thead">
                <tr>
                  <th scope="col">ID</th>
                  <th scope="col">Type</th>
                  <th scope="col">Timestamp</th>
                  <th scope="col">Details</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in history.history.slice().reverse()" v-bind:key="item.id">
                  <td scope="row">{{ item.id }}</td>
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
  </b-container>
</template>

<script lang="ts">
import { Component, Prop, Provide, Vue } from 'vue-property-decorator';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import { api, EnduroCollectionClient } from '../client';

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionWorkflow extends Vue {

  private collection: any = {};
  private history: any = {history: []};
  private activities: any = {};
  private name: string = '';

  private mounted() {
    this.loadCollection();
    this.loadHistory();
  }

  private reload() {
    this.loadHistory();
  }

  private getName(body: api.CollectionShowResponseBody): string {
    if (body.name) {
      return body.name;
    }

    return body.id.toString();
  }

  private loadCollection() {
    EnduroCollectionClient.collectionShow({id: +this.$route.params.id}).then((response: api.CollectionShowResponseBody) => {
      this.collection = response;
      this.name = this.getName(response);
    });
  }

  private loadHistory() {
    return EnduroCollectionClient.collectionWorkflow({id: +this.$route.params.id}).then((response: api.CollectionWorkflowResponseBody) => {
      this.history = response;
      this.processHistory();
    });
  }

  private processHistory() {
    const ignoredActivities = [
      'internalSessionCreationActivity',
      'internalSessionCompletionActivity',
    ];
    for (const event of this.history.history) {
      const details = event.details;
      if (event.type === 'ActivityTaskScheduled') {
        const attrs = details.activityTaskScheduledEventAttributes;
        const name = attrs.activityType.name;
        if (ignoredActivities.includes(name)) {
          continue;
        }
        this.activities[event.id] = {
          name,
          status: 'in progress',
          attempts: 0,
          started: details.timestamp,
        };
      } else if (event.type === 'ActivityTaskStarted') {
        const attrs = details.activityTaskStartedEventAttributes;
        if (attrs.scheduledEventId in this.activities) {
          const item = this.activities[attrs.scheduledEventId];
          item.attempts = attrs.attempt + 1;
        }
      } else if (event.type === 'ActivityTaskFailed') {
        const attrs = details.activityTaskFailedEventAttributes;
        if (attrs.scheduledEventId in this.activities) {
          const item = this.activities[attrs.scheduledEventId];
          item.status = 'error';
          item.completed = details.timestamp;
          item.duration = (item.completed - item.started) / 1000000000;
          item.duration = item.duration.toFixed(2);
        }
      } else if (event.type === 'ActivityTaskCompleted') {
        const attrs = details.activityTaskCompletedEventAttributes;
        if (attrs.scheduledEventId in this.activities) {
          const item = this.activities[attrs.scheduledEventId];
          item.status = 'done';
          item.completed = details.timestamp;
          item.duration = (item.completed - item.started) / 1000000000;
          item.duration = item.duration.toFixed(2);
        }
      }
    }
  }

  private renderDetails(event: api.EnduroCollectionWorkflowHistoryResponseBody): string {
    let ret = '';

    if (!event || !event.type) {
      return ret;
    }

    const attrs: any = event.details;
    if (event.type === 'ActivityTaskScheduled') {
      ret = 'Activity: ' + attrs.activityTaskScheduledEventAttributes.activityType.name;
    } else if (event.type === 'ActivityTaskFailed') {
      ret = 'Error: ' + window.atob(attrs.activityTaskFailedEventAttributes.details);
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
