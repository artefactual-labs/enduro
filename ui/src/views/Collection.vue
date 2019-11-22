<template>
  <b-container>
    <en-collection-detail/>
  </b-container>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import CollectionDetail from '@/components/CollectionDetail.vue';

@Component({
  components: {
    'en-collection-detail': CollectionDetail,
  },
})
export default class Collection extends Vue {}
</script>
