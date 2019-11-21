<template>
  <div class="collection-detail">
    <b-breadcrumb>
      <b-breadcrumb-item :to="{ name: 'collections' }">Collections</b-breadcrumb-item>
      <b-breadcrumb-item>{{ getName() }}</b-breadcrumb-item>
    </b-breadcrumb>
    <div class="container-fluid">

      <div class="row">

        <!-- Main column -->
        <div class="col-sm">
          <dl>
            <dt>Status</dt>
            <dd>
              <en-collection-status-badge :status="collection.status"/>
            </dd>
            <template v-if="collection.originalId">
              <dt>OriginalID</dt>
              <dd>{{ collection.originalId }}</dd>
            </template>
            <dt>Created</dt>
            <dd>
              {{ collection.createdAt | formatDateTime }}
            </dd>
            <template v-if="collection.transferId">
              <dt>Transfer</dt>
              <dd>{{ collection.transferId }}</dd>
            </template>
            <template v-if="collection.pipelineId">
              <dt>Pipeline</dt>
              <dd>{{ collection.pipelineId }}</dd>
            </template>
            <template v-if="collection.status == 'done'">
              <hr />
              <dt>AIP</dt>
              <dd>{{ collection.aipId }}</dd>
              <dt>Completed</dt>
              <dd>
                {{ collection.completedAt | formatDateTime }}
                (took {{ took(collection.createdAt, collection.completedAt) }})
              </dd>
            </template>
          </dl>
          <hr />
          <b-button size="sm" variant="light" v-if="!isRunning()" v-on:click="onDelete()">Delete</b-button>
        </div>

        <!-- Sidebar -->
        <div class="col-sm">
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
          <br />
          <b-card class="api">
            <b-card-title>API</b-card-title>
            <b-card-text>
              <pre class=".pre-scrollable">{{ collection }}</pre>
            </b-card-text>
          </b-card>
        </div>

      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { api, EnduroCollectionClient } from '../client';
import { Component, Inject, Prop, Provide, Vue } from 'vue-property-decorator';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import moment from 'moment';

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionDetail extends Vue {
  private interval: number = 0;
  private collection: any = {};

  private mounted() {
    this.interval = setInterval(() => this.populate(), 500);
    this.populate();
  }

  private beforeDestroy() {
    clearInterval(this.interval);
  }

  private populate() {
    EnduroCollectionClient.collectionShow({id: +this.$route.params.id}).then((response: api.CollectionShowResponseBody) => {
      this.collection = response;
    });
  }

  private getName(): string {
    let ret = this.collection.name;
    if (!ret) {
      ret = this.$route.params.id;
    }
    return ret;
  }

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

  private isRunning() {
    return !['done', 'error'].includes(this.collection.status);
  }

  private onDelete() {
    this.delete(this.collection.id).then(() => {
      this.$router.push({name: 'collections'});
    });
  }
}
</script>

<style scoped lang="scss">

.api pre { font-size: 11px; }

</style>
