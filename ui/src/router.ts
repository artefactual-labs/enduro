import Vue from 'vue';
import Router from 'vue-router';
import Collections from './views/Collections.vue';
import CollectionList from './views/CollectionList.vue';
import Collection from './views/Collection.vue';
import CollectionShow from './views/CollectionShow.vue';
import CollectionShowWorkflow from './views/CollectionShowWorkflow.vue';

Vue.use(Router);

export default new Router({
  mode: 'hash',
  base: process.env.BASE_URL,
  routes: [
    {
      path: '*',
      redirect: '/',
    },
    {
      path: '/',
      name: 'home',
      redirect: '/collections',
    },
    {
      path: '/collections',
      component: Collections,
      children: [
        {
          path: '',
          name: 'collections',
          component: CollectionList,
        },
        {
          path: '/collections/:id',
          component: Collection,
          children: [
            {
              path: '/collections/:id',
              name: 'collection',
              component: CollectionShow,
            },
            {
              path: '/collections/:id/workflow',
              name: 'collection-workflow',
              component: CollectionShowWorkflow,
            },
          ],
        },
      ],
    },
  ],
});
