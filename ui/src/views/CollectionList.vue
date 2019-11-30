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

      <div class="my-3">
        <b-form @submit="onSubmit" @reset="onReset">
          <div class="form-row">
            <div class="form-group col-12 col-sm-6 col-md-2">
              <b-form-select size="sm" v-model="status">
                <option :value="null">Status</option>
                <option value="error">Error</option>
                <option value="done">Done</option>
                <option value="in progress">In progress</option>
              </b-form-select>
            </div>
            <div class="form-group col-12 col-sm-6 col-md-2">
              <b-form-select size="sm" v-model="date">
                <option :value="null">Creation date</option>
                <option value="3h">Last 3 hours</option>
                <option value="6h">Last 6 hours</option>
                <option value="24h">Last 24 hours</option>
                <option value="3d">Last 3 days</option>
                <option value="14d">Last 14 days</option>
                <option value="30d">Last 30 days</option>
              </b-form-select>
            </div>
            <div class="form-group col-12 col-md-5 col-lg-6">
              <b-input-group size="sm">
                <b-form-input autofocus :state="$store.state.collection.validQuery" @keydown.esc="onReset" v-model="query" type="text" placeholder="Search"/>
                <b-form-select v-model="field" size="sm" class="collection-field-select">
                  <option value="name">Name</option>
                  <option value="pipeline_id">Pipeline ID</option>
                  <option value="transfer_id">Transfer ID</option>
                  <option value="aip_id">AIP ID</option>
                  <option value="original_id">Original ID</option>
                </b-form-select>
              </b-input-group>
            </div>
            <div class="form-group col-12 col-md-3 col-lg-2">
              <b-input-group size="sm">
                <b-input-group-prepend>
                  <b-button type="submit" variant="info">Search</b-button>
                </b-input-group-prepend>
                <b-input-group-append>
                  <b-button type="reset">Reset</b-button>
                </b-input-group-append>
              </b-input-group>
            </div>
          </div>
        </b-form>
      </div>

      <template v-if="results.length > 0">
        <table class="table table-bordered table-hover table-sm">
          <thead class="thead">
            <tr>
              <th scope="col">ID</th>
              <th scope="col">Name</th>
              <th scope="col">Created</th>
              <th scope="col">Completed</th>
              <th scope="col">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in results" v-bind:key="item.id" @click="rowClicked(item.id)">
              <th scope="row">{{ item.id }}</th>
              <td class="collection-name">{{ item.name || 'N/A' }}</td>
              <td>{{ item.createdAt | formatDateTime }}</td>
              <td>{{ item.completedAt | formatDateTime }}</td>
              <td><en-collection-status-badge :status="item.status"/></td>
            </tr>
          </tbody>
        </table>
        <nav>
          <ul class="pagination">
            <li class="page-item">
              <button class="page-link" @click="reloadButtonClicked">Reload</button>
            </li>
            <li class="page-item" v-if="nextCursor">
              <button class="page-link" @click="nextButtonClicked(nextCursor)">Next</button>
            </li>
          </ul>
        </nav>
      </template>
      <div v-if="results.length === 0">
        No results.
      </div>

    </template>

  </b-container>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { namespace } from 'vuex-class';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import * as CollectionStore from '../store/collection';

const collectionStoreNs = namespace('collection');

@Component({
  components: {
    'en-collection-status-badge': CollectionStatusBadge,
  },
})
export default class CollectionList extends Vue {

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_ERROR)
  private error?: boolean;

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULTS)
  private results: any;

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_NEXT_CURSOR)
  private nextCursor: any;

  @collectionStoreNs.Mutation(CollectionStore.SET_STATUS_FILTER)
  private setStatusFilter: any;

  @collectionStoreNs.Mutation(CollectionStore.SET_DATE_FILTER)
  private setDateFilter: any;

  @collectionStoreNs.Mutation(CollectionStore.SET_FIELD_FILTER)
  private setFieldFilter: any;

  @collectionStoreNs.Mutation(CollectionStore.SET_QUERY)
  private setQuery: any;

  @collectionStoreNs.Mutation(CollectionStore.SET_SEARCH_RESET)
  private reset: any;

  @collectionStoreNs.Action(CollectionStore.SEARCH_COLLECTIONS)
  private search: any;

  private created() {
    this.search();
  }

  private get status(): string | null {
    return this.$store.state.collection.query.status;
  }
  private set status(status: string | null) {
    this.setStatusFilter(status);
    this.search();
  }

  private get date(): string | null {
    return this.$store.state.collection.date;
  }
  private set date(date: string | null) {
    this.setDateFilter(date);
    this.search();
  }

  private get query(): string {
    return this.$store.state.collection.query.query;
  }
  private set query(query: string) {
    this.setQuery(query);
  }

  private get field(): string {
    return this.$store.state.collection.query.field;
  }
  private set field(field: string) {
    this.setFieldFilter(field);
  }

  /**
   * Perform same search re-using all existing state.
   */
  private retryButtonClicked() {
    this.search();
  }

  /**
   * Perform search with the cursor reset.
   */
  private reloadButtonClicked() {
    this.search();
  }

  /*/
   * Perform search with a new cursor.
   */
  private nextButtonClicked(cursor: string) {
    this.search({cursor});
  }

  /**
   * Perform search with the cursor reset.
   */
  private onSubmit(event: Event) {
    event.preventDefault();
    this.search();
  }

  /**
   * Perform search with both the query and cursor reset.
   */
  private onReset(event: Event) {
    event.preventDefault();
    this.reset();
    this.search();
  }

  /**
   * Forward user to the collection route.
   */
  private rowClicked(id: string) {
    this.$router.push({ name: 'collection', params: {id} });
  }
}
</script>

<style scoped lang="scss">

.table {
  font-size: .9rem;
}

.table thead th {
  border-bottom: 0;
}

.table th, .table td {
  cursor: pointer;
}

.table td a:hover,
.table td a:active {
  text-decoration: none !important;
}

.table-hover tbody tr:hover {
  background-color: lighten($enduro-c1, 70%);
}

.collection-name {
  color: #007bff;
}

.collection-field-select {
  max-width: 150px;
}

</style>

