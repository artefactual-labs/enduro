<template>
  <div class="collection-storage">
    <template v-if="collection">
      <b-alert :variant="summaryVariant" show>
        {{ summaryText }}
      </b-alert>

      <dl>
        <template v-if="collection.aipId">
          <dt>AIP</dt>
          <dd>{{ collection.aipId }}</dd>
        </template>

        <template v-if="collection.reconciliationStatus">
          <dt>Reconciliation status</dt>
          <dd>
            <b-badge :variant="statusVariant">{{ collection.reconciliationStatus.toUpperCase() }}</b-badge>
          </dd>
        </template>

        <template v-if="collection.aipStoredAt">
          <dt>Primary AIP stored at</dt>
          <dd>{{ collection.aipStoredAt | formatDateTime }}</dd>
        </template>

        <template v-if="collection.reconciliationCheckedAt">
          <dt>Last checked at</dt>
          <dd>{{ collection.reconciliationCheckedAt | formatDateTime }}</dd>
        </template>

        <template v-if="collection.reconciliationError">
          <dt>Reconciliation error</dt>
          <dd class="text-danger">{{ collection.reconciliationError }}</dd>
        </template>
      </dl>
    </template>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import { namespace } from 'vuex-class';
import * as CollectionStore from '../store/collection';
import { api } from '../client';

const collectionStoreNs = namespace('collection');

@Component({})
export default class CollectionShowStorage extends Vue {
  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULT)
  private collection: api.EnduroDetailedStoredCollection | undefined;

  private get summaryText(): string {
    if (!this.collection || !this.collection.aipId) {
      return 'Storage reconciliation is not available yet. An AIP has not been created.';
    }

    switch (this.collection.reconciliationStatus) {
      case 'complete':
        return 'Storage is complete.';
      case 'partial':
        return 'Primary AIP exists, but required storage is incomplete.';
      case 'unknown':
        return 'Storage state could not be determined.';
      case 'pending':
        return 'Storage reconciliation has not produced a final result yet.';
      default:
        return 'No storage reconciliation has been recorded for this collection yet.';
    }
  }

  private get summaryVariant(): string {
    if (!this.collection || !this.collection.aipId) {
      return 'secondary';
    }

    switch (this.collection.reconciliationStatus) {
      case 'complete':
        return 'success';
      case 'partial':
      case 'unknown':
        return 'warning';
      default:
        return 'secondary';
    }
  }

  private get statusVariant(): string {
    switch (this.collection?.reconciliationStatus) {
      case 'complete':
        return 'success';
      case 'partial':
        return 'warning';
      case 'unknown':
        return 'danger';
      default:
        return 'secondary';
    }
  }
}
</script>
