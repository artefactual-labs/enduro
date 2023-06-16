import Vue from 'vue';
import Vuex, { ActionTree, MutationTree, StoreOptions, Action } from 'vuex';

import { collection } from './collection';
import { pipeline } from './pipeline';
import { EnduroCollectionClient, api } from '@/client';

Vue.use(Vuex);

// Actions.
export const LOOK_UP_VERSION = 'LOOK_UP_VERSION';

// Mutations.
export const SET_VERSION = 'SET_VERSION';

export interface RootState {
  version: string;
}

const actions: ActionTree<RootState, RootState> = {

  [LOOK_UP_VERSION]({ commit }) {
    // TODO: hit a static asset?
    EnduroCollectionClient.collectionListRaw({cursor: '0'}).then((response) => {
      const v = response.raw.headers.get('X-Enduro-Version');
      commit(SET_VERSION, v);
    });
  },

};

const mutations: MutationTree<RootState> = {

  [SET_VERSION](state, version: string) {
    state.version = version;
  },

};

const store: StoreOptions<RootState> = {
  strict: import.meta.env.DEV,
  modules: {
    pipeline,
    collection,
  },

  state: { version: '' },
  actions,
  mutations,
};

export default new Vuex.Store<RootState>(store);
