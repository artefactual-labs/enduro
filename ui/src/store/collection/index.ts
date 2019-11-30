import { ActionTree, GetterTree, Module, MutationTree } from 'vuex';
import { RootState } from '@/store';

import validator from 'validator';
import moment from 'moment';

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
export const SET_STATUS_FILTER = 'SET_STATUS_FILTER';
export const SET_DATE_FILTER = 'SET_DATE_FILTER';
export const SET_FIELD_FILTER = 'SET_FIELD_FILTER';
export const SET_QUERY = 'SET_QUERY';
export const SET_SEARCH_RESET = 'SET_SEARCH_RESET';
export const SET_SEARCH_VALID = 'SET_SEARCH_VALID';

// Action types.
export const SEARCH_COLLECTION = 'SEARCH_COLLECTION';
export const SEARCH_COLLECTION_RESET = 'SEARCH_COLLECTION_RESET';
export const SEARCH_COLLECTIONS = 'SEARCH_COLLECTIONS';

const namespaced: boolean = true;

interface Query {
  status: string | null;
  earliestCreatedTime: Date | null;
  latestCreatedTime: Date | null;
  query: string;
  field: string | null;
}

interface State {
  query: Query;
  date: string | null;
  error: boolean;
  validQuery: boolean | null;
  result: any;
  results: any;
  nextCursor: string;
}

const getters: GetterTree<State, RootState> = {

  [GET_SEARCH_ERROR](state): boolean {
    return state.error;
  },

  [GET_SEARCH_RESULT](state): string {
    return state.result;
  },

  [GET_SEARCH_RESULTS](state): string {
    return state.results;
  },

  [GET_SEARCH_NEXT_CURSOR](state): string {
    return state.nextCursor;
  },

};

const actions: ActionTree<State, RootState> = {

  [SEARCH_COLLECTIONS]({ commit, state }, params): any {
    // Perform validation.
    if (
      state.query.field
      && ['pipeline_id', 'transfer_id', 'aip_id'].includes(state.query.field)
      && !validator.isUUID(state.query.query)
    ) {
      commit(SET_SEARCH_VALID, false);
      return;
    }
    commit(SET_SEARCH_VALID, null);

    const request: api.CollectionListRequest = {
      ...(params && params.cursor ? { cursor: params.cursor } : {}),
      ...(state.query.earliestCreatedTime ? { earliestCreatedTime: state.query.earliestCreatedTime } : {}),
      ...(state.query.latestCreatedTime ? { latestCreatedTime: state.query.latestCreatedTime } : {}),
    };

    // TODO: s.query.status should natively use the enum values.
    switch (state.query.status) {
      case 'new':
        request.status = api.CollectionListStatusEnum.New;
        break;
      case 'in progress':
        request.status = api.CollectionListStatusEnum.InProgress;
        break;
      case 'done':
        request.status = api.CollectionListStatusEnum.Done;
        break;
      case 'error':
        request.status = api.CollectionListStatusEnum.Error;
        break;
      case 'unknown':
        request.status = api.CollectionListStatusEnum.Unknown;
        break;
    }

    if (state.query.query) {
      switch (state.query.field) {
        case 'name':
          request.name = state.query.query;
          break;
        case 'pipeline_id':
          request.pipelineId = state.query.query;
          break;
        case 'transfer_id':
          request.transferId = state.query.query;
          break;
        case 'aip_id':
          request.aipId = state.query.query;
          break;
        case 'original_id':
          request.originalId = state.query.query;
          break;
      }
    }

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
    const request: api.CollectionShowRequest = { id: +id };
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

  [SET_SEARCH_ERROR](state, failed: boolean) {
    state.error = failed;
  },

  [SET_SEARCH_RESULT](state, result: any) {
    state.result = result;
  },

  [SET_SEARCH_RESULTS](state, results: any) {
    state.results = results;
  },

  [SET_SEARCH_NEXT_CURSOR](state, nextCursor: string) {
    state.nextCursor = nextCursor;
  },

  [SET_STATUS_FILTER](state, status: string | null) {
    state.query.status = status;
  },

  [SET_DATE_FILTER](state, date: string | null) {
    switch (date) {
      default:
      case null:
        state.date = null;
        state.query.earliestCreatedTime = null;
        state.query.latestCreatedTime = null;
        break;
      case '3h':
      case '6h':
      case '24h':
      case '3d':
      case '7d':
      case '14d':
      case '30d':
        const regex = /(\d+)([dh])/;
        const res = regex.exec(date);
        if (res && res.length === 3) {
          const unit = res[2] === 'h' ? 'hours' : 'days';
          state.query.earliestCreatedTime = moment().subtract(res[1], unit).startOf('day').utc().toDate();
          state.query.latestCreatedTime = null;
          state.date = date;
        }
    }
  },

  [SET_FIELD_FILTER](state, field: string | null) {
    state.query.field = field;
  },

  [SET_QUERY](state, query: string) {
    state.query.query = query;
  },

  [SET_SEARCH_RESET](state) {
    Object.assign(state, getDefaultState());
  },

  [SET_SEARCH_VALID](state, valid: boolean) {
    state.validQuery = valid;
  },

};

const getDefaultState = (): State => {
  return {
    query: {
      status: null,
      earliestCreatedTime: null,
      latestCreatedTime: null,
      query: '',
      field: 'name',
    },
    date: null,
    error: false,
    validQuery: null,
    result: {},
    results: [],
    nextCursor: '',
  };
};

export const collection: Module<State, RootState> = {
  namespaced,
  state: getDefaultState(),
  getters,
  actions,
  mutations,
};
