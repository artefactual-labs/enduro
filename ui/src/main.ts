import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import BootstrapVue from 'bootstrap-vue';
import { EnduroCollectionClient, setUpEnduroClient } from './client';

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
  beforeMount: () => {
    setUpEnduroClient();
  },
}).$mount('#app');
