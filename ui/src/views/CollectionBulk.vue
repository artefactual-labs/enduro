<template>

  <b-container>
    <b-row class="justify-content-center">
      <b-col cols="8">

        <div class="mt-2">
          <b-link :to="{ name: 'collections' }">&laquo; Back to search</b-link>
        </div>

        <hr />

        <h3>Bulk operation</h3>

        <template v-if="isRunning">
          <p class="mb-4">Bulk operation is still running...</p>
          <pre>{{ status }}</pre>
        </template>

        <template v-else>

          <template v-if="status.closedAt">
            <b-alert show :variant="alertVariant" dismissible>
              Last run completed with status "{{ status.status }}" at {{ status.closedAt | formatDateTimeString }}.
              <template v-if="lastFailed"><br />Check out the workflow history for more details (workflow "{{ status.workflowId }}").</template>
            </b-alert>
          </template>

          <p>Start a new bulk operation.</p>

          <b-form @submit="onSubmit">

            <b-form-group label="Collection status filter" label-for="input-status" description="Select the status of the collections that you want to modify.">
              <b-form-select id="input-operation" v-model="form.status" required>
               <b-form-select-option value="error">Error</b-form-select-option>
              </b-form-select>
            </b-form-group>

            <b-form-group label="Operation" label-for="input-size" description="Type of operation to be performed.">
              <b-form-select id="input-operation" v-model="form.operation" required>
                <b-form-select-option value="retry">Retry</b-form-select-option>
                <b-form-select-option value="cancel" disabled>Cancel</b-form-select-option>
                <b-form-select-option value="abandon" disabled>Abandon</b-form-select-option>
              </b-form-select>
            </b-form-group>

            <b-form-group label="Size" label-for="input-size" description="Optional. Maximum number of collections affected.">
              <b-form-input id="input-size" v-model="form.size" type="number"></b-form-input>
            </b-form-group>

            <b-button type="submit" variant="primary">Submit</b-button>

          </b-form>

        </template>

      </b-col>
    </b-row>
  </b-container>

</template>

<script lang="ts">

import { Component, Vue } from 'vue-property-decorator';
import { api, EnduroCollectionClient } from '../client';

@Component({})
export default class CollectionBulk extends Vue {

  private form: any = {
    status: null,
    operation: null,
    size: null,
  };

  private status: api.CollectionBulkStatusResponseBody = {
    running: false,
  };

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
    return EnduroCollectionClient.collectionBulkStatus().then((response: api.CollectionBulkStatusResponseBody) => {
      this.status = response;
      if (this.isRunning) {
        const self = this;
        setTimeout(() => {
          self.loadStatus();
        }, 1000);
      }
    }).catch((response) => {
      // tslint:disable-next-line:no-console
      console.log('Bulk status query failed!', response);
      alert('Bulk status request failed!');
    });
  }

  private onSubmit(evt: Event) {
    evt.preventDefault();
    const request: api.CollectionBulkRequest = {
      bulkRequestBody: {
        operation: this.form.operation,
        status: this.form.status,
      },
    };
    if (this.form.size > 0) {
      request.bulkRequestBody.size = +this.form.size;
    }
    return EnduroCollectionClient.collectionBulk(request).then((response: api.CollectionBulkResponseBody) => {
      this.loadStatus();
    }).catch((response: Response) => {
      if (response.status === 409) {
        alert('A new bulk has started before yours, page will reload!');
        this.loadStatus();
        return;
      }
      // tslint:disable-next-line:no-console
      console.log('Bulk request failed!', response);
      alert('Bulk request failed!');
    });
  }

}

</script>

<style lang="scss">

</style>

