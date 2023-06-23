import { ActionTree, GetterTree, Module, MutationTree } from 'vuex';
import { RootState } from '@/store';

import { EnduroCollectionClient, api, EnduroPipelineClient } from '@/client';

// Getter types.
export const GET_PIPELINE_ERROR = 'GET_PIPELINE_ERROR';
export const GET_PIPELINE_RESULT = 'GET_PIPELINE_RESULT';
export const GET_SEARCH_RESULTS = 'GET_SEARCH_RESULTS';

// Mutation types.
export const SET_PIPELINE_RESULT = 'SET_PIPELINE_RESULT';
export const SET_PIPELINE_ERROR = 'SET_PIPELINE_ERROR';
export const SET_SEARCH_RESULTS = 'SET_SEARCH_RESULTS';

// Action types.
export const SEARCH_PIPELINE = 'SEARCH_PIPELINE';
export const SEARCH_PIPELINES = 'SEARCH_PIPELINES';

const namespaced: boolean = true;

interface State {
  error: boolean;
  result: any;
  results: any;
}

const getters: GetterTree<State, RootState> = {

  [GET_PIPELINE_ERROR](state): boolean {
    return state.error;
  },

  [GET_PIPELINE_RESULT](state): string {
    return state.result;
  },

  [GET_SEARCH_RESULTS](state): string {
    return state.results;
  },

};

const actions: ActionTree<State, RootState> = {

  [SEARCH_PIPELINE]({ commit }, id): Promise<any> {
    return EnduroPipelineClient.pipelineShow({ id }).then((response: api.PipelineShowResponseBody) => {
      commit(SET_PIPELINE_RESULT, response);
      commit(SET_PIPELINE_ERROR, false);
    }).catch(() => {
      commit(SET_PIPELINE_ERROR, true);
    });
  },

  [SEARCH_PIPELINES]({ commit }): Promise<any> {
    return EnduroPipelineClient.pipelineList({ status: true }).then((response) => {
      console.log(response);
      commit(SET_SEARCH_RESULTS, response);
      commit(SET_PIPELINE_ERROR, false);
    }).catch((err) => {
      commit(SET_PIPELINE_ERROR, true);
    });
  },

};

const mutations: MutationTree<State> = {

  [SET_PIPELINE_ERROR](state, failed: boolean) {
    state.error = failed;
  },

  [SET_PIPELINE_RESULT](state, result: any) {
    state.result = result;
  },

  [SET_SEARCH_RESULTS](state, results: any) {
    state.results = results;
  },

};

const getDefaultState = (): State => {
  return {
    error: false,
    result: {},
    results: [],
  };
};

export const pipeline: Module<State, RootState> = {
  namespaced,
  state: getDefaultState(),
  getters,
  actions,
  mutations,
};
