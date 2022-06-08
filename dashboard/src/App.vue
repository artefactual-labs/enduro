<script setup lang="ts">
import Header from "./components/Header.vue";
</script>

<template>
  <div class="d-flex flex-column min-vh-100">
    <Header />
    <main class="container-xxl pt-3 flex-grow-1">
      <router-view></router-view>
    </main>
  </div>
</template>
