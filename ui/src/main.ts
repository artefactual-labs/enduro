import Vue from 'vue';
import App from './App.vue';
import router from './router';
import store from './store';
import './filters';
import Footer from '@/components/Footer.vue';
import BootstrapVue from 'bootstrap-vue';
import VueMoment from 'vue-moment';
import { setUpEnduroClient, setUpEnduroMonitor } from './client';

Vue.use(BootstrapVue);
Vue.use(VueMoment);

Vue.component('en-footer', Footer);

new Vue({
  router,
  store,
  render: (h) => h(App),
  beforeCreate: () => {
    setUpEnduroClient();
  },
  created: () => {
    setUpEnduroMonitor(store);
  },
}).$mount('#app');
