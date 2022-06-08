<script setup lang="ts">
import Footer from "../components/Footer.vue";
</script>

<template>
  Welcome to Enduro!

  <ul>
    <li><router-link :to="{ name: 'packages' }">Packages</router-link></li>
  </ul>

  <Footer />
</template>
