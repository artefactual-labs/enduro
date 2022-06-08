import { defineStore, acceptHMRUpdate } from "pinia";
import { clientProviderKey, Client, api } from "../client";
import { inject } from "vue";

export const usePackageStore = defineStore("package", {
  state: () => ({
    current: null as api.CollectionShowResponseBody | null,
    current_preservation_actions:
      null as api.CollectionPreservationActionsResponseBody | null,
  }),
  actions: {
    async fetchCurrent(id: string) {
      this.reset();
      const packageId = +id;
      if (Number.isNaN(packageId)) {
        return;
      }
      const client = inject(clientProviderKey) as Client;
      client.package.collectionShow({ id: packageId }).then((payload) => {
        this.current = payload;
      });
      client.package
        .collectionPreservationActions({ id: packageId })
        .then((payload) => {
          this.current_preservation_actions = payload;
        });
    },
    reset() {
      this.current = null;
      this.current_preservation_actions = null;
    },
  },
});

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(usePackageStore, import.meta.hot));
}
