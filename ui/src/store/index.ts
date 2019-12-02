import Vue from 'vue';
import Vuex, { StoreOptions } from 'vuex';
import { collection } from './collection';

Vue.use(Vuex);

export interface RootState {
  version?: string;
}

const store: StoreOptions<RootState> = {
  state: {},
  modules: {
    collection,
  },
  strict: process.env.NODE_ENV !== 'production',
};

export default new Vuex.Store<RootState>(store);
