import Vue from 'vue';
import Router from 'vue-router';
import Home from './views/Home.vue';
import Collections from './views/Collections.vue';
import Collection from './views/Collection.vue';
import CollectionWorkflow from './views/CollectionWorkflow.vue';

Vue.use(Router);

export default new Router({
  mode: 'hash',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home,
    },
    {
      path: '/collections',
      name: 'collections',
      component: Collections,
    },
    {
      path: '/collections/:id',
      name: 'collection',
      component: Collection,
    },
    {
      path: '/collections/:id/workflow',
      name: 'collection-workflow',
      component: CollectionWorkflow,
    },
  ],
});
