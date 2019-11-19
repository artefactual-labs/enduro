<template>
  <div class="collection-status-badge">
    <b-badge :variant="variant()">{{ value() }}</b-badge>
  </div>
</template>

<script lang="ts">

import { api } from '../client';
import { Component, Vue } from 'vue-property-decorator';

@Component({
  props: {
    status: String,
  },
})
export default class CollectionStatusBadge extends Vue {

  private status?: string;

  private defvariant: string = 'secondary';

  private variants: any = {
    [api.CollectionShowResponseBodyStatusEnum.Error]: 'danger',
    [api.CollectionShowResponseBodyStatusEnum.InProgress]: 'warning',
    [api.CollectionShowResponseBodyStatusEnum.Done]: 'success',
    [api.CollectionShowResponseBodyStatusEnum.New]: 'secondary',
    [api.CollectionShowResponseBodyStatusEnum.Unknown]: 'secondary',
  };

  private variant(): string {
    if (!this.status) {
      return '';
    }

    if (this.status in this.variants) {
      return this.variants[this.status];
    }

    return this.defvariant;
  }

  private value(): string {
    if (!this.status) {
      return '';
    }

    return this.status.toUpperCase();
  }

  // TODO: extend b-badge?

  // <b-badge v-if="status == 'done'" variant="success">{{ status }}</b-badge>
  // <b-badge v-else-if="status == 'in progress'" variant="warning">{{ status }}</b-badge>
  // <b-badge v-else-if="status == 'error'" variant="danger">{{ status }}</b-badge>
  // <b-badge v-else variant="secondary">{{ status }}</b-badge>

}

</script>

<style scoped lang="scss">

.collection-status-badge {

}

</style>
