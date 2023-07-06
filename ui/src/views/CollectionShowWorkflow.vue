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
          <dd>{{ startedAt | formatDateTimeString }}</dd>
        </template>

        <template v-if="completedAt">
          <dt>Completed</dt>
          <dd>
            {{ completedAt | formatDateTimeString }}
            (took {{ startedAt | formatEpochDuration(completedAt) }})
          </dd>
        </template>

        <dt>Activity summary</dt>
        <dd>
          <b-list-group id="activity-summary">
            <b-list-group-item v-for="(item, index) in activities" v-bind:key="index" :class="{ failed: item.status === 'error' || item.status === 'timed out', local: item.local }">
              <en-collection-status-badge class="float-right" :status="item.status"/>
              <span class="name" v-if="!item.local">{{ item.name }}</span>
              <span class="name" v-else>{{ item.name }} (local activity)</span>
              <span class="date" v-if="item.started">{{ item.started | formatDateTimeString }}</span>
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
              <span class="date">{{ item.details.event_time | formatDateTimeString }}</span>
              <div v-html="historyEventDescription(item)"></div>
            </b-list-group-item>
          </b-list-group>
        </dd>

      </dl>
    </template>

  </div>
</template>

<script lang="ts">
import { Component, Watch, Vue } from 'vue-property-decorator';
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

  @Watch('collection')
  private onCollectionChanged(val: object, oldVal: object) {
    // Reload the workflow history when the collection changes.
    this.loadHistory();
  }

  private async loadHistory() {
    return EnduroCollectionClient.collectionWorkflow({id: +this.$route.params.id}).then((response: api.CollectionWorkflowResponseBody) => {
      this.history = response;
      this.processHistory();
    }).catch((err) => {
      this.error = true;

      // tslint:disable-next-line:no-console
      console.log(err);
    });
  }

  private parseEncodedField(input: string): any {
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
        const attrs = details.Attributes.marker_recorded_event_attributes;
        if (attrs.marker_name !== 'LocalActivity') {
          continue;
        }
        const dataPayloads = attrs.details.data.payloads;
        if (!dataPayloads.length) {
          continue;
        }
        const innerDetails = this.parseEncodedField(dataPayloads[0].data);
        this.activities[event.id] = {
          local: true,
          name: innerDetails.ActivityType,
          attempts: innerDetails.Attempt,
          replayed: innerDetails.ReplayTime,
        };
      } else if (event.type === 'ActivityTaskScheduled') {
        const attrs = details.Attributes.activity_task_scheduled_event_attributes;
        const name = attrs.activity_type.name;
        if (ignoredActivities.includes(name)) {
          continue;
        }
        this.activities[event.id] = {
          local: false,
          name,
          status: 'in progress',
          attempts: 0,
          started: details.event_time,
        };
      } else if (event.type === 'ActivityTaskStarted') {
        const attrs = details.Attributes.activity_task_started_event_attributes;
        if (attrs.scheduled_event_id in this.activities) {
          const item = this.activities[attrs.scheduled_event_id];
          item.attempts = attrs.attempt;
        }
      } else if (event.type === 'ActivityTaskFailed') {
        const attrs = details.Attributes.activity_task_failed_event_attributes;
        this.activityError = true;
        if (attrs.scheduled_event_id in this.activities) {
          const item = this.activities[attrs.scheduled_event_id];
          item.status = 'error';
          item.details = 'Message: ' + attrs.failure.message;
          item.completed = details.event_time;
          item.duration = this.duration(item.started, item.completed);
        }
      } else if (event.type === 'ActivityTaskCompleted') {
        const attrs = details.Attributes.activity_task_completed_event_attributes;
        if (attrs.scheduled_event_id in this.activities) {
          const item = this.activities[attrs.scheduled_event_id];
          item.status = 'done';
          item.completed = details.event_time;
          item.duration = this.duration(item.started, item.completed);
          if (item.name === 'async-completion-activity' && attrs.result) {
            item.details = 'User selection: ' + window.atob(attrs.result) + '.';
          }
        }
      } else if (event.type === 'ActivityTaskTimedOut') {
        const attrs = details.Attributes.activity_task_timed_out_event_attributes;
        if (attrs.scheduled_event_id in this.activities) {
          const item = this.activities[attrs.scheduled_event_id];
          item.status = 'timed out';
          item.details = 'Timeout ' + attrs.timeout_type + '.';
        }
      } else if (event.type === 'WorkflowExecutionStarted') {
        this.startedAt = details.event_time;
      } else if (event.type === 'WorkflowExecutionCompleted') {
        this.completedAt = details.event_time;
      } else if (event.type === 'WorkflowExecutionFailed') {
        const attrs = details.Attributes.workflow_execution_failed_event_attributes;
        this.workflowError = this.workflowErrorDescription(attrs.failure);
        this.completedAt = details.event_time;
      }
    }
  }

  private workflowErrorDescription(failure: any): string {
    let desc = '';
    if (failure.hasOwnProperty('message')) {
      desc = failure.message;
    }
    if (failure.hasOwnProperty('cause') && failure.cause.hasOwnProperty('message')) {
      if (desc.length) desc += ': ';
      desc += failure.cause.message;
    }
    return desc
  }

  private duration(startedAt: string, completedAt: string): string {
    const started = new Date(startedAt);
    const completed = new Date(completedAt);
    const took = (completed.getTime() - started.getTime()) / 1000;
    return took.toLocaleString();
  }

  private historyEventDescription(event: api.EnduroCollectionWorkflowHistoryResponseBody): string {
    let ret = '';

    if (!event || !event.type) {
      return ret;
    }

    const attrs: any = event.details;

    if (event.type === 'ActivityTaskScheduled') {
      ret = 'Activity: ' + attrs.Attributes.activity_task_scheduled_event_attributes.activity_type.name;
    } else if (event.type === 'ActivityTaskFailed') {
      const body = attrs.Attributes.activity_task_failed_event_attributes;
      ret = JSON.stringify(body, null, 2);
    } else if (event.type === 'DecisionTaskScheduled') {
      const attempt: number = parseInt(attrs.decisionTaskScheduledEventAttributes.attempt, 10) + 1;
      ret = 'Attempts: ' + attempt;
    } else if (event.type === 'WorkflowExecutionFailed') {
      const body = attrs.Attributes.workflow_execution_failed_event_attributes;
      ret = JSON.stringify(body, null, 2);
    } else if (event.type == 'WorkflowExecutionStarted') {
      const body = attrs.Attributes.workflow_execution_started_event_attributes;
      ret = JSON.stringify(body, null, 2);
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
      display: block;
      color: #666;
      padding: 0.5rem 0;
    }
    .failed .details {
      margin-top: .5rem;
      border-top: 2px solid #ff4545;
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
