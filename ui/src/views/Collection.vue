<template>
  <CollectionDetail></CollectionDetail>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import CollectionDetail from '@/components/CollectionDetail.vue';

@Component({
  components: {
    CollectionDetail,
  },
})
export default class Collection extends Vue {}
</script>
