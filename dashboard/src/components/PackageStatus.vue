<script setup lang="ts">
const props = defineProps({
  status: { type: String, required: true },
});
</script>

<template>{{ props.status }}</template>
