import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import BootstrapVue from 'bootstrap-vue';
import * as client from './client/src';

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

console.log(apiPath());

export let EnduroCollectionClient = new client.CollectionApi(
  new client.Configuration({basePath: apiPath()},
));

Vue.use(BootstrapVue);

Vue.filter('formatDateTime', (value: string) => {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  return date.toLocaleString();
});

Vue.filter('formatEpoch', (value: number) => {
  if (!value) {
    return '';
  }
  const date = new Date(value / 1000 / 1000); // TODO: is this right?
  return date.toLocaleString();
});

new Vue({
  router,
  store,
  render: (h) => h(App),
}).$mount('#app');
