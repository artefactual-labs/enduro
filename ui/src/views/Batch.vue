<template>

  <b-container>
    <b-row class="justify-content-center">
      <b-col cols="8">

        <div class="mt-2">
          <b-link :to="{ name: 'collections' }">&laquo; Back to search</b-link>
        </div>

        <hr />

        <h3>Batch</h3>

        <template v-if="isRunning">
          <p class="mb-4">Batch operation is still running...</p>
          <pre>{{ status }}</pre>
        </template>

        <template v-else>

          <p>Start a new batch.</p>

          <b-form @submit="onSubmit">

            <b-form-group label="Path" label-for="input-path" description="Select the path for the batch.">
              <b-form-input id="input-path" v-model="form.path" type="text" required></b-form-input>
            </b-form-group>

            <pipeline-dropdown v-on:pipeline-selected="onPipelineSelected($event)"/>

            <pipeline-processing-configuration-dropdown :pipeline-id="pipelineId" v-on:pipeline-processing-configuration-selected="form.processingConfig = $event"/>

            <div class="actions">
              <b-button type="submit" variant="primary">Submit</b-button>
            </div>

          </b-form>

        </template>

      </b-col>
    </b-row>
  </b-container>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { api, EnduroBatchClient } from '../client';
import PipelineDropdown from '@/components/PipelineDropdown.vue';
import PipelineProcessingConfigurationDropdown from '@/components/PipelineProcessingConfigurationDropdown.vue';

@Component({
  components: {
    PipelineDropdown,
    PipelineProcessingConfigurationDropdown,
  },
})
export default class Batch extends Vue {

  private form: any = {
    path: null,
    pipeline: null,
    processingConfig: null,
  };

  private pipelineId: string = '';

  private status: api.BatchStatusResponseBody = {
    running: false,
  };

  private onPipelineSelected($event: any): void {
    this.pipelineId = $event ? $event.value : null;
    this.form.pipeline = $event ? $event.text : null;
  }

  private get alertVariant() {
    return this.status.status === 'completed' ? 'success' : 'warning';
  }

  private get lastFailed() {
    return this.status.status !== 'completed';
  }

  private get isRunning() {
    return this.status && this.status.running;
  }

  private created() {
    this.loadStatus();
  }

  private loadStatus() {
    return EnduroBatchClient.batchStatus().then((response: api.BatchStatusResponseBody) => {
      this.status = response;
      if (this.isRunning) {
        const self = this;
        setTimeout(() => {
          self.loadStatus();
        }, 1000);
      }
    }).catch((response) => {
      // tslint:disable-next-line:no-console
      console.log('Batch status query failed!', response);
      alert('Batch status request failed!');
    });
  }

  private onSubmit(evt: Event) {
    evt.preventDefault();
    const request: api.BatchSubmitRequest = {
      submitRequestBody: {
        path: this.form.path,
        pipeline: this.form.pipeline,
        processingConfig: this.form.processingConfig,
      },
    };
    return EnduroBatchClient.batchSubmit(request).then((response: api.BatchSubmitResponseBody) => {
      this.loadStatus();
    }).catch((response: Response) => {
      if (response.status === 409) {
        alert('A new batch has started before yours, page will reload!');
        this.loadStatus();
        return;
      }
      // tslint:disable-next-line:no-console
      console.log('Batch request failed!', response);
      alert('Batch request failed!');
    });
  }

}

</script>

<style lang="scss">

</style>

