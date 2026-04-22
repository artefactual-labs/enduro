import type { Router } from 'vue-router'
import { DataLoaderPlugin } from 'vue-router/experimental'

export default defineNuxtPlugin({
  name: 'data-loaders',
  dependsOn: ['nuxt:router'],
  setup(nuxtApp) {
    nuxtApp.vueApp.use(DataLoaderPlugin, {
      router: nuxtApp.$router as Router,
      isSSR: import.meta.server
    })
  }
})
