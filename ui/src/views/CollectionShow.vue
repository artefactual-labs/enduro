<template>

  <b-row>

    <!-- Main column -->
    <b-col cols="6">

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
          <dt>Completed</dt>
          <dd>
            {{ collection.completedAt | formatDateTime }}
            (took {{ took(collection.startedAt, collection.completedAt) }})
          </dd>
        </template>

        <template v-if="collection.transferId">
          <dt>Transfer</dt>
          <dd>{{ collection.transferId }}</dd>
        </template>

        <template v-if="collection.pipelineId">
          <dt>Pipeline</dt>
          <dd>{{ collection.pipelineId }}</dd>
        </template>

        <template v-if="collection.status == 'done' && collection.aipId">
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

      <b-card>
        <b-card-title>Workflow</b-card-title>
        <b-card-text>
          <dl>
            <dt>ID</dt>
            <dd>{{ collection.workflowId }}</dd>
            <dt>RunID</dt>
            <dd>{{ collection.runId }}</dd>
          </dl>
          <small>Use Cadence's Web UI for more details.</small>
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
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import moment from 'moment';

import { api, EnduroCollectionClient } from '../client';

const collectionStoreNs = namespace('collection');

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionShow extends Vue {

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULT)
  private collection: any;

  @collectionStoreNs.Action(CollectionStore.SEARCH_COLLECTION)
  private search: any;

  private took(created: Date, completed: Date): string {
    const diff = moment(completed).diff(created);
    return moment.duration(diff).humanize();
  }

  private retry(id: string): Promise<any> {
    return EnduroCollectionClient.collectionRetry({id: +id});
  }

  private cancel(id: string): Promise<any> {
    return EnduroCollectionClient.collectionCancel({id: +id});
  }

  private delete(id: string): Promise<any> {
    return EnduroCollectionClient.collectionDelete({id: +id});
  }

  private get isRunning() {
    return !['done', 'error'].includes(this.collection.status);
  }

  private onDelete() {
    this.delete(this.collection.id).then(() => {
      this.$router.push({name: 'collections'});
    });
  }

  private onDeleteConfirmed() {
    this.onDelete();
  }

  private onReload() {
    this.search(this.collection.id);
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
