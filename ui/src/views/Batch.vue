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

          <b-form @submit.prevent="onSubmit">

            <b-form-group label="Path" label-for="input-path" description="Select the path for the batch.">
              <b-form-input id="input-path" v-model="form.path" type="text" required></b-form-input>
            </b-form-group>

            <pipeline-dropdown v-on:pipeline-selected="onPipelineSelected($event)"/>

            <pipeline-processing-configuration-dropdown v-show="pipelineId" :pipeline-id="pipelineId" v-on:pipeline-processing-configuration-selected="form.processingConfig = $event"/>

            <b-form-group label="Transfer type" label-for="dropdown-select" description="Optional. Choose the transfer type, with the default being standard.">
              <b-form-select id="dropdown-select" v-model="form.transferType" :options="transferOptions"></b-form-select>
            </b-form-group>

            <b-form-group label-for="reject-duplicates-checkbox">
              <b-form-checkbox id="reject-duplicates-checkbox" v-model="form.rejectDuplicates">
                Reject transfers with duplicate names.
              </b-form-checkbox>
            </b-form-group>

            <b-form-group label-for="process-name-metadata-checkbox">
              <b-form-checkbox id="process-name-metadata-checkbox" v-model="form.processNameMetadata">
                Process transfer name metadata.
              </b-form-checkbox>
            </b-form-group>

            <b-tabs content-class="mt-3" tite-item-class="mt-3" v-model="tabIndex">
              <b-tab title="Completed directory" active>
                <div class="form-group">
                  <input v-model="form.completedDir" type="text" class="form-control" id="completed-directory-input" aria-describedby="completed-directory-help">
                  <small id="completed-directory-help" class="form-text text-muted">
                    Optional. The path where transfers are moved into when processing has completed successfully.
                    <p v-if="hints.completedDirs">
                      Known directories:
                      <ul>
                        <li v-for="item in hints.completedDirs">
                          <a href="#" @click.prevent="form.completedDir = item">{{ item }}</a>
                        </li>
                      </ul>
                    </p>
                  </small>
                </div>
              </b-tab>
              <b-tab title="Retention period">
                <div class="form-group">
                  <input v-model="form.retentionPeriod" type="text" class="form-control" id="retention-period-input" aria-describedby="retention-period-help">
                  <small id="retention-period-help" class="form-text text-muted">
                    Optional. The duration of time for which the transfer should be retained before being removed. The string should be constructed as a sequence of decimal numbers, each with optional fraction and a unit suffix, such as "30m", "24h" or "2h30m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
                  </small>
                </div>
              </b-tab>
            </b-tabs>

            <div class="actions">
              <b-button type="submit" variant="primary" class="mx-2">Submit</b-button>
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

type UserDefaults = {
  transferType: string | null;
  rejectDuplicates: boolean;
  processNameMetadata: boolean;
  completedDir: string | null;
  retentionPeriod: string | null;
}

const batchDefaultsStorageKey = "batchDefaults";

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
    transferType: null,
    rejectDuplicates: false,
    processNameMetadata: false,
    completedDir: null,
    retentionPeriod: null,
  };

  private tabIndex: number = 0;

  private pipelineId: string = '';

  private status: api.BatchStatusResponseBody = {
    running: false,
  };

  private hints: api.BatchHintsResponseBody = {
    completedDirs: [],
  };

  private transferOptions = [
    { value: 'standard', text: 'Standard' },
    { value: 'zipfile', text: 'Zipfile' },
    { value: 'unzipped bag', text: 'Unzipped bag' },
    { value: 'zipped bag', text: 'Zipped bag' },
    { value: 'dspace', text: 'DSpace' },
    { value: 'maildir', text: 'Maildir' },
    { value: 'TRIM', text: 'TRIM' },
    { value: 'dataverse', text: 'Dataverse' },
  ];

  private onPipelineSelected($event: any): void {
    this.pipelineId = $event ? $event.value : null;
    this.form.pipeline = $event ? $event.text : null;
  }

  private get isRunning() {
    return this.status && this.status.running;
  }

  private created() {
    this.loadDefaults();
    this.loadStatus();
    this.loadHints();
  }

  // Load form defaults from previous choices made by the user.
  private loadDefaults() {
    const batchDefaults = localStorage.getItem(batchDefaultsStorageKey);
    if (batchDefaults === null ) {
      return;
    }
    let defaults : UserDefaults;
    try {
      defaults = JSON.parse(batchDefaults);
    } catch(e) {
      localStorage.removeItem(batchDefaultsStorageKey);
      return;
    }
    this.form.transferType = defaults.transferType;
    this.form.rejectDuplicates = defaults.rejectDuplicates;
    this.form.processNameMetadata = defaults.processNameMetadata;
    this.form.completedDir = defaults.completedDir;
    this.form.retentionPeriod = defaults.retentionPeriod;
  }

  // Save choices made by the user that can be used as defaults next time.
  private saveDefaults() {
    const defaults: UserDefaults = {
       transferType: this.form.transferType,
       rejectDuplicates: this.form.rejectDuplicates,
       processNameMetadata: this.form.processNameMetadata,
       completedDir: this.form.completedDir,
       retentionPeriod: this.form.retentionPeriod,
    };
    localStorage.setItem(batchDefaultsStorageKey, JSON.stringify(defaults));
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

  private loadHints() {
    return EnduroBatchClient.batchHints().then((response: api.BatchHintsResponseBody) => {
      this.hints = response;
    });
  }

  private onSubmit(evt: Event) {
    const request: api.BatchSubmitRequest = {
      submitRequestBody: {
        path: this.form.path
      },
    };
    if (this.form.pipeline) {
      request.submitRequestBody.pipeline = this.form.pipeline;
    }
    if (this.form.processingConfig) {
      request.submitRequestBody.processingConfig = this.form.processingConfig;
    }
    if (this.form.completedDir && this.tabIndex === 0) {
      request.submitRequestBody.completedDir = this.form.completedDir;
    }
    if (this.form.retentionPeriod && this.tabIndex === 1) {
      request.submitRequestBody.retentionPeriod = this.form.retentionPeriod;
    }
    request.submitRequestBody.rejectDuplicates = this.form.rejectDuplicates;
    if (this.form.transferType) {
      request.submitRequestBody.transferType = this.form.transferType;
    }
    request.submitRequestBody.processNameMetadata = this.form.processNameMetadata;
    return EnduroBatchClient.batchSubmit(request).then((response: api.BatchSubmitResponseBody) => {
      this.saveDefaults();
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

