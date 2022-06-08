<script setup lang="ts">
import { useRoute } from "vue-router";
import { usePackageStore } from "../../stores/package";

const route = useRoute();
const packageStore = usePackageStore();

packageStore.fetchCurrent(route.params.id.toString());
</script>

<template>
  <div>
    <div class="row" v-if="packageStore.current">
      <div class="col-12">
        <nav aria-label="breadcrumb">
          <ol class="breadcrumb">
            <li class="breadcrumb-item">
              <router-link :to="{ name: 'packages' }">Packages</router-link>
            </li>
            <li class="breadcrumb-item active" aria-current="page">
              {{ packageStore.current.name }}
            </li>
          </ol>
        </nav>
      </div>
      <div class="col-12">
        <ul class="nav nav-tabs">
          <li class="nav-item">
            <router-link
              class="nav-link"
              exact-active-class="active"
              :to="{
                name: 'packages-id',
                params: { id: packageStore.current.id },
              }"
              >Overview</router-link
            >
          </li>
          <li class="nav-item">
            <router-link
              class="nav-link"
              exact-active-class="active"
              :to="{
                name: 'packages-id-workflow',
                params: { id: packageStore.current.id },
              }"
              >Workflow</router-link
            >
          </li>
        </ul>
      </div>
    </div>
    <router-view></router-view>
  </div>
</template>
