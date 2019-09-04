<template>
  <div class="collection-list">
    <table class="table table-bordered table-hover">
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
  </div>
</template>

<script lang="ts">
import { Component, Prop, Provide, Vue } from 'vue-property-decorator';
import { EnduroCollectionClient } from '../main';
import CollectionStatusBadge from '@/components/CollectionStatusBadge.vue';
import { CollectionEnduroStoredCollectionResponseCollection } from '../client/src';

@Component({
  components: {
    CollectionStatusBadge,
  },
})
export default class CollectionList extends Vue {

  private interval: number = 0;
  private collections: object[] = [];

  private created() {
    this.interval = setInterval(() => this.populate(), 500);
    return this.populate();
  }

  private beforeDestroy() {
    clearInterval(this.interval);
  }

  private populate() {
    return EnduroCollectionClient.collectionList({}).then((response: CollectionEnduroStoredCollectionResponseCollection) => {
      this.collections = response;
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
  font-size: 1rem;
}

.table thead th {
  border-bottom: 0;
}

.table th, .table td {
  padding: 0.50rem;
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
