import { ActionTree, GetterTree, Module, MutationTree } from 'vuex';
import { RootState } from '@/store';

import validator from 'validator';
import moment from 'moment';
import { debounce } from 'lodash';

import { EnduroCollectionClient, api } from '@/client';

// Getter types.
export const GET_SEARCH_ERROR = 'GET_SEARCH_ERROR';
export const GET_SEARCH_RESULT = 'GET_SEARCH_RESULT';
export const GET_SEARCH_RESULTS = 'GET_SEARCH_RESULTS';
export const SHOW_PREV_LINK = 'SHOW_PREV_LINK';
export const SHOW_NEXT_LINK = 'SHOW_NEXT_LINK';

// Mutation types.
export const SET_SEARCH_RESULT = 'SET_SEARCH_RESULT';
export const SET_SEARCH_RESULTS = 'SET_SEARCH_RESULTS';
export const SET_SEARCH_ERROR = 'SET_SEARCH_ERROR';
export const SET_STATUS_FILTER = 'SET_STATUS_FILTER';
export const SET_DATE_FILTER = 'SET_DATE_FILTER';
export const SET_FIELD_FILTER = 'SET_FIELD_FILTER';
export const SET_QUERY = 'SET_QUERY';
export const SET_SEARCH_RESET = 'SET_SEARCH_RESET';
export const SET_SEARCH_VALID = 'SET_SEARCH_VALID';
export const SET_SEARCH_HOME_PAGE = 'SET_SEARCH_HOME_PAGE';
export const SET_SEARCH_PREV_PAGE = 'SET_SEARCH_PREV_PAGE';
export const SET_SEARCH_NEXT_PAGE = 'SET_SEARCH_NEXT_PAGE';
export const SET_SEARCH_NEXT_CURSOR = 'SET_SEARCH_NEXT_CURSOR';
export const SET_WORKFLOW_DECISION_ERROR = 'SET_WORKFLOW_DECISION_ERROR';

// Action types.
export const SEARCH_COLLECTION = 'SEARCH_COLLECTION';
export const SEARCH_COLLECTION_RESET = 'SEARCH_COLLECTION_RESET';
export const SEARCH_COLLECTIONS = 'SEARCH_COLLECTIONS';
export const SEARCH_COLLECTIONS_DEBOUNCED = 'SEARCH_COLLECTIONS_DEBOUNCED';
export const SEARCH_COLLECTIONS_HOME_PAGE = 'SEARCH_COLLECTIONS_HOME_PAGE';
export const SEARCH_COLLECTIONS_PREV_PAGE = 'SEARCH_COLLECTIONS_PREV_PAGE';
export const SEARCH_COLLECTIONS_NEXT_PAGE = 'SEARCH_COLLECTIONS_NEXT_PAGE';
export const MAKE_WORKFLOW_DECISION = 'MAKE_WORKFLOW_DECISION';
export const ON_SOCKET_MESSAGE = 'ON_SOCKET_MESSAGE';

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

  seenCursors: Array<string | null>;
  currCursor: string | null;
  prevCursor: string | null;
  nextCursor: string | null;
}

function resetCursors(state: State) {
  state.seenCursors = [];
  state.currCursor = '';
  state.prevCursor = null;
  state.nextCursor = null;
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

  [SHOW_PREV_LINK](state): boolean {
    return state.prevCursor !== null;
  },

  [SHOW_NEXT_LINK](state): boolean {
    return state.nextCursor !== null;
  },

};

const debouncedSearch = debounce((dispatch) => {
  dispatch(SEARCH_COLLECTIONS);
}, 1000);

const actions: ActionTree<State, RootState> = {

  [SEARCH_COLLECTION]({ commit, dispatch }, id): any {
    const request: api.CollectionShowRequest = { id: +id };
    return EnduroCollectionClient.collectionShow(request).then((response) => {
      commit(SET_SEARCH_RESULT, response);
      commit(SET_SEARCH_ERROR, false);
      dispatch('pipeline/SEARCH_PIPELINE', response.pipelineId, { root: true });
    }).catch(() => {
      commit(SET_SEARCH_ERROR, true);
    });
  },

  [SEARCH_COLLECTION_RESET]({ commit }) {
    commit(SET_SEARCH_RESULT, {});
    commit(SET_SEARCH_ERROR, false);
  },

  [SEARCH_COLLECTIONS]({ commit, state }): any {
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
      ...(state.currCursor && state.currCursor ? { cursor: state.currCursor } : {}),
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
      case 'queued':
        request.status = api.CollectionListStatusEnum.Queued;
        break;
      case 'pending':
        request.status = api.CollectionListStatusEnum.Pending;
        break;
      case 'abandoned':
        request.status = api.CollectionListStatusEnum.Abandoned;
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

    return EnduroCollectionClient.collectionList(request).then((response) => {
      // collectionList does not transform the objects as collectionShow does.
      // Do it manually for now, I'd expect the generated client to start doing
      // this for us at some point.
      response.items = response.items.map((item) => api.EnduroStoredCollectionFromJSON(item));

      commit(SET_SEARCH_RESULTS, response.items);
      commit(SET_SEARCH_ERROR, false);
      commit(SET_SEARCH_NEXT_CURSOR, response.nextCursor);
    }).catch(() => {
      commit(SET_SEARCH_ERROR, true);
    });
  },

  [SEARCH_COLLECTIONS_DEBOUNCED]({ dispatch }): any {
    debouncedSearch(dispatch);
  },

  [SEARCH_COLLECTIONS_HOME_PAGE]({ commit, dispatch }): any {
    commit(SET_SEARCH_HOME_PAGE);
    dispatch(SEARCH_COLLECTIONS);
  },

  [SEARCH_COLLECTIONS_PREV_PAGE]({ commit, dispatch }): any {
    commit(SET_SEARCH_PREV_PAGE);
    dispatch(SEARCH_COLLECTIONS);
  },

  [SEARCH_COLLECTIONS_NEXT_PAGE]({ commit, dispatch }): any {
    commit(SET_SEARCH_NEXT_PAGE);
    dispatch(SEARCH_COLLECTIONS);
  },

  [MAKE_WORKFLOW_DECISION]({ commit }, payload): any {
    const request: api.CollectionDecideOperationRequest = {
      id: +payload.id,
      collectionDecideRequest: {
        option: payload.option,
      },
    };
    return EnduroCollectionClient.collectionDecide(request).then(() => {
      commit(SET_WORKFLOW_DECISION_ERROR, false);
    }).catch(() => {
      commit(SET_WORKFLOW_DECISION_ERROR, true);
    });
  },

  [ON_SOCKET_MESSAGE]({ state, dispatch }, payload): void {
    const event = JSON.parse(payload.event.data);

    // Update global collection search. We use the debounced action to limit
    // the number of API requests we send to the server.
    if (['collection:created', 'collection:updated'].includes(event.type)) {
      dispatch(SEARCH_COLLECTIONS_DEBOUNCED);
    }

    // Update current collection if necessary.
    if (event.type === 'collection:updated') {
      const current = state.result;
      if (Object.keys(current).length > 0 && current.id === event.id) {
        dispatch(SEARCH_COLLECTION, event.id);
      }
    }
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

  [SET_STATUS_FILTER](state, status: string | null) {
    state.query.status = status;
    resetCursors(state);
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

    resetCursors(state);
  },

  [SET_FIELD_FILTER](state, field: string | null) {
    state.query.field = field;
    resetCursors(state);
  },

  [SET_QUERY](state, query: string) {
    state.query.query = query;
    resetCursors(state);
  },

  [SET_SEARCH_RESET](state) {
    Object.assign(state, getDefaultState());
  },

  [SET_SEARCH_VALID](state, valid: boolean) {
    state.validQuery = valid;
  },

  [SET_SEARCH_HOME_PAGE](state) {
    resetCursors(state);
  },

  [SET_SEARCH_PREV_PAGE](state) {
    const tail = state.seenCursors.pop();
    if (typeof(tail) === 'undefined') {
      return;
    }
    state.currCursor = tail;
    state.prevCursor = state.seenCursors.length ? state.seenCursors[state.seenCursors.length - 1] : null;
    state.nextCursor = null;
  },

  [SET_SEARCH_NEXT_PAGE](state) {
    state.seenCursors.push(state.currCursor);
    state.prevCursor = state.currCursor;
    state.currCursor = state.nextCursor;
  },

  [SET_SEARCH_NEXT_CURSOR](state, cursor: string | undefined) {
    state.nextCursor = cursor ? cursor : null;
  },

  [SET_WORKFLOW_DECISION_ERROR](state, failed: boolean) {
    state.error = true;
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

    // Cursor-based pagination.
    seenCursors: [],
    currCursor: '',
    prevCursor: null,
    nextCursor: null,
  };
};

export const collection: Module<State, RootState> = {
  namespaced,
  state: getDefaultState(),
  getters,
  actions,
  mutations,
};
