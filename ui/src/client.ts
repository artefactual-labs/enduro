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

function setUpEnduroClient() {
  let path = apiPath();

  // path seems to be wrong when Enduro is deployed behind FortiNet SSLVPN.
  // There is some URL rewriting going on that is beyond my understanding.
  // This is an attempt to rewrite the URL using their url_rewrite function.
  if (typeof window.fgt_sslvpn !== 'undefined') {
    path = window.fgt_sslvpn.url_rewrite(path);
  }

  const config: api.Configuration = new api.Configuration({basePath: path});
  EnduroCollectionClient = new api.CollectionApi(config);
  EnduroPipelineClient = new api.PipelineApi(config);

  // tslint:disable-next-line:no-console
  console.log('Enduro client created', path);
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
  setUpEnduroClient,
  api,
};
