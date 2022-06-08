import { createApp } from "vue";
import { createRouter, createWebHistory } from "vue-router";
import routes from "~pages";
import { createClient, clientProviderKey, api } from "./client";
import App from "./App.vue";
import { createPinia } from "pinia";
import moment from "moment";
import humanizeDuration from "humanize-duration";

import "./styles/main.scss";

const router = createRouter({
  history: createWebHistory("/"),
  routes,
  strict: false,
});

const client = createClient();
const pinia = createPinia();

const app = createApp(App);
app.use(router);
app.use(pinia);
app.mount("#app");
app.provide(clientProviderKey, client);

interface Filters {
  [key: string]: (value: any) => string;
}

declare module "@vue/runtime-core" {
  interface ComponentCustomProperties {
    $filters: Filters;
  }
}

app.config.globalProperties.$filters = {
  formatDateTimeString(value: string) {
    const date = new Date(value);
    return date.toLocaleString();
  },
  formatDateTime(value: Date | undefined) {
    if (!value) {
      return "";
    }
    return value.toLocaleString();
  },
  formatDuration(from: Date, to: Date) {
    const diff = moment(to).diff(from);
    return humanizeDuration(moment.duration(diff).asMilliseconds());
  },
  formatPreservationActionStatus(
    value: api.EnduroCollectionPreservationActionsActionResponseBodyStatusEnum
  ) {
    switch (value) {
      case api.EnduroCollectionPreservationActionsActionResponseBodyStatusEnum
        .Complete:
        return "bg-success";
      case api.EnduroCollectionPreservationActionsActionResponseBodyStatusEnum
        .Failed:
        return "bg-danger";
      case api.EnduroCollectionPreservationActionsActionResponseBodyStatusEnum
        .Processing:
        return "bg-warning";
      default:
        return "bg-secondary";
    }
  },
};
