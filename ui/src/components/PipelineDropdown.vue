<template>
  <div class="form-group">

    <label for="pipeline-dropdown">Pipeline</label>

    <select
      class="form-control"
      id="pipeline-dropdown"
      @change="onChange">

      <option selected value>Select a pipeline</option>

      <option
        v-for="item in options"
        v-bind:value="item.value">
          {{ item.text }}
      </option>

    </select>

    <small
      id="pipeline-dropdown-help"
      class="form-text text-muted">
        Optional. Choose one of the pipelines configured.
    </small>

  </div>
</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { api, EnduroPipelineClient } from '../client';

@Component({})
export default class PipelineDropdown extends Vue {

  private options: Array<{ text: string, value: string }> = [];

  private mounted(): void {
    EnduroPipelineClient.pipelineList({}).then((response) => {
      response.forEach((element) => {
        if (element.id === undefined) {
          return;
        }
        this.options.push({text: element.name, value: element.id});
      });
    });
  }

  // Emit the option or null.
  private onChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const option = this.options.filter((item) => item.value === select.value);
    this.$emit('pipeline-selected', option.length ? option[0] : null);
  }

}

</script>
