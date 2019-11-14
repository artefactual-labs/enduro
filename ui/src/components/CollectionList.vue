<template>
  <div class="collection-list">
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
        <tr v-for="item in collections" v-bind:key="item.id" @click="view(item.id)">
          <th scope="row">{{ item.id }}</th>
          <td class="collection-name">{{ item.name }}</td>
          <td>{{ item.created_at | formatDateTime }}</td>
          <td>{{ item.completed_at | formatDateTime }}</td>
          <td>
            <CollectionStatusBadge :status="item.status"/>
          </td>
        </tr>
      </tbody>
    </table>
    <nav>
      <ul class="pagination">
        <li class="page-item">
          <button class="page-link" @click="next('')">Reload</button>
        </li>
        <li class="page-item" v-if="nextCursor">
          <button class="page-link" @click="next(nextCursor)">Next</button>
        </li>
      </ul>
    </nav>
  </div>
</template>

<script lang="ts">
import { Component, Prop, Provide, Vue } from 'vue-property-decorator';
import { EnduroCollectionClient } from '../client';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import { CollectionListRequest, CollectionListResponseBody, EnduroStoredCollectionResponseBodyCollection } from '../client/src';

@Component({
  components: {
    CollectionStatusBadge,
  },
})
export default class CollectionList extends Vue {

  private interval: number = 0;
  private collections: EnduroStoredCollectionResponseBodyCollection = [];
  private nextCursor?: string = '';

  private created() {
    this.next();
  }

  private load(cursor?: string) {
    const request: CollectionListRequest = {};
    if (cursor) {
      request.cursor = cursor;
    }
    return EnduroCollectionClient.collectionList(request);
  }

  private next(cursor?: string) {
    this.load(cursor).then((response: CollectionListResponseBody) => {
      this.collections = response.items;
      this.nextCursor = response.nextCursor;
    });
  }

  private view(id: string) {
    this.$router.push({
      name: 'collection',
      params: {id},
    });
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
    background-color: #17a2b820;
}

.collection-name {
  color: #007bff;
}

</style>
