import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import './filters';
import BootstrapVue from 'bootstrap-vue';
import VueMoment from 'vue-moment';
import { setUpEnduroClient } from './client';

Vue.use(BootstrapVue);
Vue.use(VueMoment);

new Vue({
  router,
  store,
  render: (h) => h(App),
  beforeMount: () => {
    setUpEnduroClient();
  },
}).$mount('#app');
