<template>
  <div class="collection-detail">

    <template v-if="error">
      <p>The workflow history is not available.</p>
    </template>

    <template v-else>
      <b-link class="reload" v-on:click="reload()">Reload</b-link>

      <b-alert variant="danger" v-if="workflowError" show>
        <h4 class="alert-heading">Workflow failure</h4>
        <p>The workflow completed with an error, see below for more details.</p>
        <hr /><pre>{{ workflowError }}</pre>
      </b-alert>

      <b-alert variant="warning" v-else-if="activityError" show>
        <h4 class="alert-heading">Activity error(s)</h4>
        At least one activity has failed, see below for more details.
      </b-alert>

      <dl>

        <dt>ID</dt>
        <dd>{{ collection.workflowId }}</dd>

        <dt>Instance (RunID)</dt>
        <dd>{{ collection.runId }}</dd>

        <dt>Status</dt>
        <dd>
          <b-badge>{{ history.status }}</b-badge>
        </dd>

        <template v-if="startedAt">
          <dt>Started</dt>
          <dd>{{ startedAt | formatEpoch }}</dd>
        </template>

        <template v-if="completedAt">
          <dt>Completed</dt>
          <dd>
            {{ completedAt | formatEpoch }}
            (took {{ startedAt | formatEpochDuration(completedAt) }})
          </dd>
        </template>

        <dt>Activity summary</dt>
        <dd>
          <b-list-group id="activity-summary">
            <b-list-group-item v-for="(item, index) in activities" v-bind:key="index" :class="{ failed: item.status === 'error', local: item.local }">
              <en-collection-status-badge class="float-right" :status="item.status"/>
              <span class="name" v-if="!item.local">{{ item.name }}</span>
              <span class="name" v-else>{{ item.name }} (local activity)</span>
              <span class="date" v-if="item.started">{{ item.started | formatEpoch }}</span>
              <span class="date" v-if="item.replayed">{{ item.replayed | formatDateTimeString }}</span>
              <span class="duration float-right" v-if="item.duration">{{ item.duration }}s</span>
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
import { namespace } from 'vuex-class';
import * as CollectionStore from '../store/collection';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import { api, EnduroCollectionClient } from '../client';

const collectionStoreNs = namespace('collection');

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionShowWorkflow extends Vue {

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULT)
  private collection: any;

  private history: any = {history: []};
  private activities: any = {};
  private name: string = '';
  private error: boolean = false;
  private workflowError: string = '';
  private activityError: boolean = false;
  private startedAt: string = '';
  private completedAt: string = '';

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

  private parseEncodedField(input: string): string {
    const value = window.atob(input);
    try {
      return JSON.parse(value);
    } catch {
      return value;
    }
  }

  private processHistory() {
    const ignoredActivities = [
      'internalSessionCreationActivity',
      'internalSessionCompletionActivity',
    ];
    for (const event of this.history.history) {
      const details = event.details;
      if (event.type === 'MarkerRecorded') {
        const attrs = details.markerRecordedEventAttributes;
        if (attrs.markerName === 'LocalActivity') {
          const innerDetails = JSON.parse(window.atob(attrs.details));
          this.activities[event.id] = {
            local: true,
            name: innerDetails.activityType,
            attempts: 0,
            replayed: innerDetails.replayTime,
          };
          if (innerDetails.hasOwnProperty('resultJson')) {
            // JSON.parse(innerDetails.resultJson);
          }
        }
      } else if (event.type === 'ActivityTaskScheduled') {
        const attrs = details.activityTaskScheduledEventAttributes;
        const name = attrs.activityType.name;
        if (ignoredActivities.includes(name)) {
          continue;
        }
        this.activities[event.id] = {
          local: false,
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
        this.activityError = true;
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
      } else if (event.type === 'WorkflowExecutionStarted') {
        this.startedAt = details.timestamp;
      } else if (event.type === 'WorkflowExecutionCompleted') {
        this.completedAt = details.timestamp;
      } else if (event.type === 'WorkflowExecutionFailed') {
        const attrs = details.workflowExecutionFailedEventAttributes;
        const reason = attrs.reason;
        const info = this.parseEncodedField(attrs.details);
        this.workflowError = reason + ' - ' + info;
        this.completedAt = details.timestamp;
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
    } else if (event.type === 'WorkflowExecutionFailed') {
      const reason = attrs.workflowExecutionFailedEventAttributes.reason;
      const info = this.parseEncodedField(attrs.workflowExecutionFailedEventAttributes.details);
      ret = reason + ' - ' + info;
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
      &.local {
        background: rgb(238,238,238);
        background: linear-gradient(90deg, rgba(255, 255, 255, 1) 75%, rgba(238, 238, 238, 1) 100%);
        font-size: 0.75rem;
        padding: 0.25rem 0.75rem;
        .name {
          font-weight: normal;
        }
        .date {
          display: none;
        }
      }
    }
    .name {
      display: block;
      font-weight: bold;
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
