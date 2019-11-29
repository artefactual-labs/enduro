import { ActionTree, GetterTree, Module, MutationTree } from 'vuex';
import { RootState } from '@/store';

import { EnduroCollectionClient, api } from '@/client';

// Getter types.
export const GET_SEARCH_ERROR = 'GET_SEARCH_ERROR';
export const GET_SEARCH_RESULT = 'GET_SEARCH_RESULT';
export const GET_SEARCH_RESULTS = 'GET_SEARCH_RESULTS';
export const GET_SEARCH_NEXT_CURSOR = 'GET_SEARCH_NEXT_CURSOR';

// Mutation types.
export const SET_SEARCH_RESULT = 'SET_SEARCH_RESULT';
export const SET_SEARCH_RESULTS = 'SET_SEARCH_RESULTS';
export const SET_SEARCH_ERROR = 'SET_SEARCH_ERROR';
export const SET_SEARCH_NEXT_CURSOR = 'SET_SEARCH_NEXT_CURSOR';

// Action types.
export const SEARCH_COLLECTION = 'SEARCH_COLLECTION';
export const SEARCH_COLLECTION_RESET = 'SEARCH_COLLECTION_RESET';
export const SEARCH_COLLECTIONS = 'SEARCH_COLLECTIONS';

const namespaced: boolean = true;

interface State {
  error: boolean;
  result: any;
  results: any;
  nextCursor: string;
}

const state: State = {
  error: false,
  result: {},
  results: [],
  nextCursor: '',
};

const getters: GetterTree<State, RootState> = {

  [GET_SEARCH_ERROR](s): boolean {
    return s.error;
  },

  [GET_SEARCH_RESULT](s): string {
    return s.result;
  },

  [GET_SEARCH_RESULTS](s): string {
    return s.results;
  },

  [GET_SEARCH_NEXT_CURSOR](s): string {
    return s.nextCursor;
  },

};

const actions: ActionTree<State, RootState> = {

  [SEARCH_COLLECTIONS]({ commit }, params): any {
    const request: api.CollectionListRequest = {
      ...(params && params.cursor ? { cursor: params.cursor } : {}),
    };
    return EnduroCollectionClient.collectionList(request).then((response: api.CollectionListResponseBody) => {
      // collectionList does not transform the objects as collectionShow does.
      // Do it manually for now, I'd expect the generated client to start doing
      // this for us at some point.
      response.items = response.items.map(
        (item: api.EnduroStoredCollectionResponseBody): api.EnduroStoredCollectionResponseBody =>
          api.EnduroStoredCollectionResponseBodyFromJSON(item),
      );
      commit(SET_SEARCH_RESULTS, response.items);
      commit(SET_SEARCH_NEXT_CURSOR, response.nextCursor);
      commit(SET_SEARCH_ERROR, false);
    }).catch(() => {
      commit(SET_SEARCH_ERROR, true);
    });
  },

  [SEARCH_COLLECTION]({ commit }, id): any {
    const request: api.CollectionShowRequest = {id: +id};
    return EnduroCollectionClient.collectionShow(request).then((response: api.CollectionShowResponseBody) => {
      commit(SET_SEARCH_RESULT, response);
      commit(SET_SEARCH_ERROR, false);
    }).catch(() => {
      commit(SET_SEARCH_ERROR, true);
    });
  },

  [SEARCH_COLLECTION_RESET]({ commit }) {
    commit(SET_SEARCH_RESULT, {});
    commit(SET_SEARCH_ERROR, false);
  },

};

const mutations: MutationTree<State> = {

  [SET_SEARCH_ERROR](s, failed: boolean) {
    s.error = failed;
  },

  [SET_SEARCH_RESULT](s, result: any) {
    s.result = result;
  },

  [SET_SEARCH_RESULTS](s, results: any) {
    s.results = results;
  },

  [SET_SEARCH_NEXT_CURSOR](s, nextCursor: string) {
    s.nextCursor = nextCursor;
  },

};

export const collection: Module<State, RootState> = {
  namespaced,
  state,
  getters,
  actions,
  mutations,
};
