<template>
  <footer class="mt-5">
    <b-container class="py-4">
      Enduro is licensed under Apache License 2.0.<br />
      © {{ date }} Artefactual Systems, Inc.
      <p>{{ version }}</p>
    </b-container>
  </footer>
</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { Action } from 'vuex-class';
import store, { LOOK_UP_VERSION } from '../store';

@Component({
  props: {
    status: String,
  },
})
export default class Footer extends Vue {

  @Action(LOOK_UP_VERSION)
  private lookUp: any;

  private get date(): number {
    return new Date().getFullYear();
  }

  private get version(): string {
    return this.$store.state.version;
  }

  private created() {
    this.lookUp();
  }
}

</script>

<style scoped lang="scss">

  p {
    font-size: 1.2rem;
    margin-top: 1rem;
  }

</style>
