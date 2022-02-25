<template>

  <b-row>

    <!-- Main column -->
    <b-col cols="6">

      <b-alert variant="warning" v-if="isPending" show>
        <h4 class="alert-heading">Awaiting decision</h4>
        <p>An activity has failed irremediably. More information can be found under the <b-link :to="{ name: 'collection-workflow', params: {id: $route.params.id} }">Workflow</b-link> tab. You can decide what to do next.</p>
        <b-button-group size="sm">
          <b-button @click="onDecisionAbandon()">Abandon</b-button>
          <b-button @click="onDecisionRetry()" variant="info">Retry</b-button>
        </b-button-group>
      </b-alert>

      <dl>

        <dt>Status</dt>
        <dd><en-collection-status-badge :status="collection.status"/></dd>

        <template v-if="collection.originalId">
          <dt>OriginalID</dt>
          <dd>{{ collection.originalId }}</dd>
        </template>

        <dt>Created</dt>
        <dd>
          {{ collection.createdAt | formatDateTime }}
        </dd>

        <template>
          <dt>Started</dt>
          <dd v-if="collection.startedAt">
            {{ collection.startedAt | formatDateTime }}
          </dd>
          <dd v-else>
            Not started yet.
          </dd>
        </template>

        <template v-if="collection.status == 'done'">
          <dt>Stored</dt>
          <dd>
            {{ collection.completedAt | formatDateTime }}
            (took {{ collection.startedAt | formatDuration(collection.completedAt) }})
          </dd>
        </template>

        <template v-if="collection.transferId">
          <dt>Transfer</dt>
          <dd>{{ collection.transferId }}</dd>
        </template>

        <template v-if="collection.aipId">
          <dt>AIP</dt>
          <dd>{{ collection.aipId }}</dd>
        </template>

      </dl>

      <div class="actions mt-5">

        <b-link class="small" v-if="!isRunning" v-b-modal="'delete-modal'">Delete</b-link>
        <b-modal id="delete-modal" @ok="onDeleteConfirmed()" title="Are you sure?">
          Once completed, this operation cannot be reversed.
          <template v-slot:modal-footer="{ ok, cancel, hide }">
            <b-button size="sm" variant="danger" @click="ok()">Delete</b-button>
            <b-button size="sm" variant="light" @click="cancel()">Cancel</b-button>
          </template>
        </b-modal>
        <span class="divider">|</span>
        <b-link class="small" @click="onReload()">Reload</b-link>

      </div>

    </b-col>

    <!-- Sidebar -->
    <b-col cols="6">

      <b-card class="mb-4">
        <b-card-title v-if="pipeline.name">Pipeline {{ pipeline.name }}</b-card-title>
        <b-card-title v-else>Pipeline</b-card-title>

        <b-card-text v-if="collection.pipelineId">
          {{ collection.pipelineId }}<br />
          <small>Usage: {{ pipeline.current }} of {{ pipeline.capacity }} slots.</small>
        </b-card-text>
        <b-card-text v-else>
          Not identified yet.
        </b-card-text>
      </b-card>

      <b-card>
        <b-card-title>Workflow</b-card-title>
        <b-card-text>
          <dl>
            <dt>ID</dt>
            <dd>{{ collection.workflowId }}</dd>
            <dt>RunID</dt>
            <dd>{{ collection.runId }}</dd>
          </dl>
          <hr />
          <b-button-group size="sm">
            <b-button :to="{ name: 'collection-workflow', items: {id: collection.id} }">Status</b-button>
            <b-button v-if="collection.status == 'error'" variant="info" v-on:click="retry(collection.id)">Retry</b-button>
            <b-button v-if="collection.status == 'in progress'" variant="dark" v-on:click="cancel(collection.id)">Cancel</b-button>
          </b-button-group>
        </b-card-text>
      </b-card>

    </b-col>

  </b-row>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { namespace } from 'vuex-class';
import * as CollectionStore from '../store/collection';
import * as PipelineStore from '../store/pipeline';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';

import { api, EnduroCollectionClient } from '../client';

const collectionStoreNs = namespace('collection');
const pipelineStoreNs = namespace('pipeline');

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionShow extends Vue {

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULT)
  private collection: api.CollectionShowResponseBody | undefined;

  @pipelineStoreNs.Getter(PipelineStore.GET_PIPELINE_RESULT)
  private pipeline: api.PipelineShowResponseBody | undefined;

  @collectionStoreNs.Action(CollectionStore.SEARCH_COLLECTION)
  private search: any;

  @collectionStoreNs.Action(CollectionStore.MAKE_WORKFLOW_DECISION)
  private decide: any;

  private retry(id: number): Promise<any> {
    return EnduroCollectionClient.collectionRetry({id});
  }

  private cancel(id: number): Promise<any> {
    return EnduroCollectionClient.collectionCancel({id});
  }

  private delete(id: number): Promise<any> {
    return EnduroCollectionClient.collectionDelete({id});
  }

  private get isRunning() {
    if (!this.collection) {
      return false;
    }
    return ['new', 'in progress', 'queued', 'pending'].includes(this.collection.status);
  }

  private get isPending() {
    if (!this.collection) {
      return false;
    }
    return ['pending'].includes(this.collection.status);
  }

  private onDelete() {
    if (!this.collection) {
      return;
    }
    this.delete(+this.collection.id).then(() => {
      this.$router.push({name: 'collections'});
    });
  }

  private onDeleteConfirmed() {
    this.onDelete();
  }

  private onReload() {
    if (!this.collection) {
      return;
    }
    this.search(this.collection.id);
  }

  private onDecisionAbandon() {
    if (!this.collection) {
      return;
    }
    this.decide({id: +this.collection.id, option: 'ABANDON'});
  }

  private onDecisionRetry() {
    if (!this.collection) {
      return;
    }
    this.decide({id: +this.collection.id, option: 'RETRY_ONCE'});
  }

}

</script>


<style scoped lang="scss">

.actions {
  .divider {
    display: inline-block;
    text-align: center;
    width: 1rem;
    color: $enduro-bolight;
  }
}

</style>
