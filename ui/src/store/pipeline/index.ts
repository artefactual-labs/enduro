import { ActionTree, GetterTree, Module, MutationTree } from 'vuex';
import { RootState } from '@/store';

import { EnduroCollectionClient, api, EnduroPipelineClient } from '@/client';

// Getter types.
export const GET_PIPELINE_ERROR = 'GET_PIPELINE_ERROR';
export const GET_PIPELINE_RESULT = 'GET_PIPELINE_RESULT';

// Mutation types.
export const SET_PIPELINE_RESULT = 'SET_PIPELINE_RESULT';
export const SET_PIPELINE_ERROR = 'SET_PIPELINE_ERROR';

// Action types.
export const SEARCH_PIPELINE = 'SEARCH_PIPELINE';

const namespaced: boolean = true;

interface State {
  error: boolean;
  result: any;
}

const getters: GetterTree<State, RootState> = {

  [GET_PIPELINE_ERROR](state): boolean {
    return state.error;
  },

  [GET_PIPELINE_RESULT](state): string {
    return state.result;
  },

};

const actions: ActionTree<State, RootState> = {

  [SEARCH_PIPELINE]({ commit }, id): any {
    const request: api.PipelineShowRequest = { id };
    return EnduroPipelineClient.pipelineShow(request).then((response: api.PipelineShowResponseBody) => {
    commit(SET_PIPELINE_RESULT, response);
    commit(SET_PIPELINE_ERROR, false);
    }).catch(() => {
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

};

const getDefaultState = (): State => {
  return {
    error: false,
    result: {},
  };
};

export const pipeline: Module<State, RootState> = {
  namespaced,
  state: getDefaultState(),
  getters,
  actions,
  mutations,
};
