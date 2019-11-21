<template>
  <div class="collections">
    <en-collection-list/>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import CollectionList from '@/components/CollectionList.vue';

@Component({
  components: {
    'en-collection-list': CollectionList,
  },
})

export default class Collections extends Vue {}
</script>
