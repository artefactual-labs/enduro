<template>
  <div class="form-group">

    <label for="pipeline-processing-config-dropdown">Processing configuration</label>

    <select
      class="form-control"
      id="pipeline-processing-configuration-dropdown"
      described-by="pipeline-processing-configuration-dropdown-help"
      v-on:change="$emit('pipeline-processing-configuration-selected', $event.target.value)"
      required>

      <option selected value>Select a processing configuration</option>

      <option
        v-for="item in options"
        v-bind:value="item.value">
          {{ item.text }}
      </option>

    </select>

    <small
      id="pipeline-processing-configuration-dropdown-help"
      class="form-text text-muted">
        Choose one of the processing configurations available.
    </small>

  </div>
</template>

<script lang="ts">

import { Component, Vue, Watch } from 'vue-property-decorator';
import { api, EnduroPipelineClient } from '../client';

@Component({
  props: {
    pipelineId: {
      type: String,
      required: false,
    },
  },
})
export default class PipelineProcessingConfigurationDropdown extends Vue {

  private options: Array<{ text: string, value: string }> = [];

  @Watch('pipelineId')
  private onPipelineChanged(val: string, oldVal: string) {
    this.options = [];
    if (!val) {
      return;
    }
    EnduroPipelineClient.pipelineProcessing({id: val}).then((response) => {
      response.forEach((name) => {
        this.options.push({text: name, value: name});
      });
    });
  }

}

</script>
