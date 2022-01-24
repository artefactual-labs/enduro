import * as api from './openapi-generator';

declare global {
  interface Window {
    enduro: any;
    fgt_sslvpn: any;
  }
}

function apiPath(): string {
  const location = window.location;
  const path = location.protocol
    + '//'
    + location.hostname
    + (location.port ? ':' + location.port : '')
    + location.pathname
    + (location.search ? location.search : '');

  return path.replace(/\/$/, '');
}

let EnduroCollectionClient: api.CollectionApi;
let EnduroPipelineClient: api.PipelineApi;
let EnduroBatchClient: api.BatchApi;

function getPath(): string {
  let path = apiPath();

  // path seems to be wrong when Enduro is deployed behind FortiNet SSLVPN.
  // There is some URL rewriting going on that is beyond my understanding.
  // This is an attempt to rewrite the URL using their url_rewrite function.
  if (typeof window.fgt_sslvpn !== 'undefined') {
    path = window.fgt_sslvpn.url_rewrite(path);
  }

  return path;
}

function getWebSocketURL(): string {
  let url = getPath();

  if (url.startsWith('https')) {
    url = 'wss' + url.slice('https'.length);
  } else if (url.startsWith('http')) {
    url = 'ws' + url.slice('http'.length);
  }

  return url;
}

function setUpEnduroClient(): void {
  const path = getPath();
  const config: api.Configuration = new api.Configuration({basePath: path});

  EnduroCollectionClient = new api.CollectionApi(config);
  EnduroPipelineClient = new api.PipelineApi(config);
  EnduroBatchClient = new api.BatchApi(config);

  // tslint:disable-next-line:no-console
  console.log('Enduro client created', path);
}

function setUpEnduroMonitor(store: any) {
  const url = getWebSocketURL() + '/collection/monitor';
  const socket = new WebSocket(url);
  socket.onmessage = (event) => {
    store.dispatch('collection/ON_SOCKET_MESSAGE', {event});
  };
  socket.onclose = (event) => {
    // tslint:disable-next-line:no-console
    console.log('Enduro WebSocket client closed', event.code);
  };

  // tslint:disable-next-line:no-console
  console.log('Enduro WebSocket client created', url);
}

window.enduro = {
  // As a last resort, user could run window.enduro.reload() from console?
  reload: () => {
    setUpEnduroClient();
  },
};

export {
  EnduroCollectionClient,
  EnduroPipelineClient,
  EnduroBatchClient,
  setUpEnduroClient,
  setUpEnduroMonitor,
  api,
};
