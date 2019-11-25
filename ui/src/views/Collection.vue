<template>

  <div>

    <b-container fluid class="collection-nav px-0">

      <!-- Breadcrumb -->
      <b-container>
        <b-breadcrumb class="py-3 px-0 mb-0">
          <b-breadcrumb-item :to="{ name: 'collections' }">Collections</b-breadcrumb-item>
          <b-breadcrumb-item :to="{ name: 'collection', params: {id: $route.params.id} }">{{ result.name || 'N/A' }}</b-breadcrumb-item>
        </b-breadcrumb>
      </b-container>

      <!-- Tabs -->
      <b-container>
        <b-nav tabs>
          <b-nav-item v-bind:active="$route.name == 'collection'" :to="{ name: 'collection', params: {id: $route.params.id} }">Overview</b-nav-item>
          <b-nav-item v-bind:active="$route.name == 'collection-workflow'" :to="{ name: 'collection-workflow', params: {id: $route.params.id} }">Workflow</b-nav-item>
        </b-nav>
      </b-container>

    </b-container>

    <!-- Body -->
    <b-container class="pt-3">
      <router-view/>
    </b-container>

  </div>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { namespace } from 'vuex-class';
import * as CollectionStore from '../store/collection';

const collectionStoreNs = namespace('collection');

@Component({})
export default class Collection extends Vue {

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_RESULT)
  private result: any;

  @collectionStoreNs.Getter(CollectionStore.GET_SEARCH_ERROR)
  private error: any;

  @collectionStoreNs.Action(CollectionStore.SEARCH_COLLECTION)
  private search: any;

  @collectionStoreNs.Action(CollectionStore.SEARCH_COLLECTION_RESET)
  private reset: any;

  private async created() {
    this.reset();
    await this.search(+this.$route.params.id);
    if (this.error) {
      this.$router.push({'name': 'collections'});
    }
  }

}

</script>

<style scoped lang="scss">

.collection-nav {
  background-color: $enduro-bglight;
  border-bottom: 1px solid #dee2e6;
  .breadcrumb {
    background-color: transparent;
  }
  .nav-tabs {
    border-bottom: 0;
  }
  .nav-link {
    border-top: 3px solid transparent;
    &.active {
      border-top: 3px solid $enduro-c1 !important;
      border-left-color: #dee2e6 !important;
      border-right-color: #dee2e6 !important;
    }
  }
  .nav-tabs .nav-link:hover,
  .nav-tabs .nav-link:focus {
    border-left-color: transparent;
    border-right-color: transparent;
  }
}

</style>
