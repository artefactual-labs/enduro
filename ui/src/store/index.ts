import Vue from 'vue';
import Vuex, { GetterTree, ActionTree, MutationTree, StoreOptions, Action } from 'vuex';

import { collection } from './collection';
import { EnduroCollectionClient, api } from '@/client';

Vue.use(Vuex);

// Getters.
export const GET_VERSION = 'GET_VERSION';

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
    EnduroCollectionClient.collectionShow({id: 0}).catch((response) => {
      const v = response.headers.get('X-Enduro-Version');
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
  strict: process.env.NODE_ENV !== 'production',
  modules: {
    collection,
  },

  state: { version: '' },
  actions,
  mutations,
};

export default new Vuex.Store<RootState>(store);
