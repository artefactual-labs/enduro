<script setup lang="ts">
import { inject, onMounted, reactive } from "vue";
import { clientProviderKey, Client, api } from "../../client";
import PackageStatus from "../../components/PackageStatus.vue";

const client = inject(clientProviderKey) as Client;

const items: Array<api.EnduroStoredCollectionResponseBody> = reactive([]);

onMounted(() => {
  client.package.collectionList().then((resp) => {
    Object.assign(items, resp.items);
  });
});
</script>

<template>
  <h2>Packages</h2>
  <table v-bind="$attrs" class="table table-striped table-hover">
    <thead>
      <tr>
        <th scope="col">ID</th>
        <th scope="col">Name</th>
        <th scope="col">UUID</th>
        <th scope="col">Started</th>
        <th scope="col">Location</th>
        <th scope="col">Status</th>
      </tr>
    </thead>
    <tbody>
      <tr v-for="pkg in items" :key="pkg.id">
        <td scope="row">{{ pkg.id }}</td>
        <td>
          <router-link :to="{ name: 'packages-id', params: { id: pkg.id } }">{{
            pkg.name
          }}</router-link>
        </td>
        <td>{{ pkg.aipId }}</td>
        <td>{{ $filters.formatDateTime(pkg.startedAt) }}</td>
        <td>Location?</td>
        <td>
          <PackageStatus :status="pkg.status" />
        </td>
      </tr>
    </tbody>
  </table>
</template>

<style scoped></style>
