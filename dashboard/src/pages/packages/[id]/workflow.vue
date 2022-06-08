<script setup lang="ts">
import { usePackageStore } from "../../../stores/package";

const packageStore = usePackageStore();
</script>

<template>
  <div>This is the workflow page</div>
  <template v-bind="$attrs">
    <div v-if="packageStore.current">
      <pre>Name: {{ packageStore.current.name }} </pre>
    </div>
    <div v-else>Loading...</div>
  </template>
</template>
