<template>
  <div class="collections">
    <CollectionList></CollectionList>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import CollectionList from '@/components/CollectionList.vue';

@Component({
  components: {
    CollectionList,
  },
})

export default class Collections extends Vue {}
</script>
