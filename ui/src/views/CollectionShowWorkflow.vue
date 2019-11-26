<template>
  <div class="collection-detail">

    <template v-if="error">
      <p>The workflow history is not available.</p>
    </template>

    <template v-else>
      <b-link class="reload" v-on:click="reload()">Reload</b-link>
      <dl>
        <dt>Status</dt>
        <dd><b-badge>{{ history.status }}</b-badge></dd>

        <dt>Activity summary</dt>
        <dd>
          <b-list-group id="activity-summary">
            <b-list-group-item v-for="(item, index) in activities" v-bind:key="index" :class="{ failed: item.status === 'error' }">
              <en-collection-status-badge class="float-right" :status="item.status"/>
              <strong>{{ item.name }}</strong><br />
              <span class="date">{{ item.started | formatEpoch }}</span>
              <span class="float-right duration">{{ item.duration }}s</span>
              <span class="attempts ml-1" v-if="item.attempts > 1">({{ item.attempts }} attempts)</span>
              <span class="details" v-if="item.details">{{ item.details }}</span>
            </b-list-group-item>
          </b-list-group>
        </dd>

        <dt>History</dt>
        <dd>
          <b-list-group id="activity-summary">
            <b-list-group-item v-for="item in history.history.slice().reverse()" v-bind:key="item.id">
              <span class="float-right identifier">#{{ item.id }}</span>
              <strong>{{ item.type }}</strong><br />
              <span class="date">{{ item.details.timestamp | formatEpoch }}</span>
              <div v-html="renderDetails(item)"></div>
            </b-list-group-item>
          </b-list-group>
        </dd>

      </dl>
    </template>

  </div>
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
export default class CollectionShowWorkflow extends Vue {

  private collection: any = {};
  private history: any = {history: []};
  private activities: any = {};
  private name: string = '';
  private error: boolean = false;

  private created() {
    this.loadHistory();
  }

  private reload() {
    this.loadHistory();
  }

  private loadHistory() {
    return EnduroCollectionClient.collectionWorkflow({id: +this.$route.params.id}).then((response: api.CollectionWorkflowResponseBody) => {
      this.history = response;
      this.processHistory();
    }).catch((response) => {
      this.error = true;
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
          item.details = window.atob(attrs.details);
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

    if (ret.length) {
      ret = '<pre>' + ret + '</pre>';
    }

    return ret;
  }

}
</script>

<style lang="scss">

.collection-detail {

  .reload {
    float: right;
  }

  dd {
    margin-bottom: 1.5rem;
  }

  #activity-summary {
    font-size: .8rem;
    .list-group-item {
      padding: 0.50rem 0.75rem;
    }
    .failed {
      background-color: #ff000011;
    }
    .details {
      border-top: 2px solid #ff4545;
      display: block;
      color: #666;
      margin-top: .5rem;
      padding: 0.5rem 0;
    }
    .date, .duration {
      color: #999;
    }
    .identifier {
      color: #999;
    }
  }

  pre {
    margin: 3px 0 0 0 !important;
    overflow: auto !important;
  }

}

</style>
