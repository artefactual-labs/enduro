<script setup lang="ts">
import { usePackageStore } from "../../../stores/package";

const packageStore = usePackageStore();
</script>

<template>
  <div v-if="packageStore.current">
    <div class="row mt-3">
      <div class="col">
        <h2>AIP creation details</h2>
        <dl>
          <dt>Name</dt>
          <dd>{{ packageStore.current.name }}</dd>
          <dt>AIP UUID</dt>
          <dd>{{ packageStore.current.aipId }}</dd>
          <dt>Workflow status</dt>
          <dd>
            <span class="badge text-bg-warning"
              >TODO: {{ packageStore.current.status }}</span
            >
            (Create and Review AIP)
          </dd>
          <dt>Started</dt>
          <dd>{{ packageStore.current.startedAt }}</dd>
        </dl>
        <pre>{{ packageStore.current }}</pre>
      </div>
      <div class="col">
        <div class="card mb-3">
          <div class="card-body">
            <h5 class="card-title">Location</h5>
            <p class="card-text">
              <a href="#">aip-review</a>
            </p>
            <div class="">
              <a href="#" class="btn btn-primary btn-sm"
                >Choose storage location</a
              >
            </div>
          </div>
        </div>
        <div class="card">
          <div class="card-body">
            <h5 class="card-title">Package details</h5>
            <dl>
              <dt>Original objects</dt>
              <dd>14</dd>
              <dt>Package size</dt>
              <dd>1.45 GB</dd>
              <dt>Last workflow outcome</dt>
              <dd>
                <span class="badge text-bg-warning"
                  >TODO: {{ packageStore.current.status }}</span
                >
                (Create and Review AIP)
              </dd>
            </dl>
            <div class="">
              <a href="#" class="btn btn-secondary btn-sm me-3"
                >View metadata summary</a
              >
              <a href="#" class="btn btn-primary btn-sm">Download</a>
            </div>
          </div>
        </div>
      </div>
    </div>
    <div class="row">
      <div class="col">
        <h2>Preservation actions</h2>
        <div class="card mb-3">
          <div class="card-body">
            <p>
              Create and Review AIP
              <span class="badge text-bg-warning"
                >TODO: {{ packageStore.current.status }}</span
              >
            </p>
            <span v-if="packageStore.current.completedAt">
              Completed
              {{ $filters.formatDateTime(packageStore.current.completedAt) }}
              (took
              {{
                $filters.formatDuration(
                  packageStore.current.startedAt,
                  packageStore.current.completedAt
                )
              }})
            </span>
          </div>
        </div>
        <table
          class="table table-bordered table-sm"
          v-if="
            packageStore.current_preservation_actions &&
            packageStore.current_preservation_actions.actions
          "
        >
          <thead>
            <tr>
              <th scope="col">Task #</th>
              <th scope="col">Name</th>
              <th scope="col">Outcome</th>
              <th scope="col">Notes</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(action, idx) in packageStore.current_preservation_actions
                .actions"
              :key="action.id"
            >
              <td>{{ idx + 1 }}</td>
              <td>{{ action.name }}</td>
              <td>
                <span
                  class="badge"
                  :class="
                    $filters.formatPreservationActionStatus(action.status)
                  "
                  >{{ action.status }}</span
                >
              </td>
              <td>TODO: note goes here</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped></style>
