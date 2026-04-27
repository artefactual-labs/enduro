import { codecovVitePlugin } from '@codecov/vite-plugin'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxt/test-utils/module'
  ],

  ssr: false,

  devtools: {
    enabled: true
  },
  app: {
    baseURL: '/'
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      enduroApiBase: import.meta.env.NUXT_PUBLIC_ENDURO_API_BASE || ''
    }
  },

  routeRules: {
    '/collection': { proxy: 'http://127.0.0.1:9000/collection' },
    '/collection/**': { proxy: 'http://127.0.0.1:9000/collection/**' },
    '/pipeline': { proxy: 'http://127.0.0.1:9000/pipeline' },
    '/pipeline/**': { proxy: 'http://127.0.0.1:9000/pipeline/**' },
    '/batch': { proxy: 'http://127.0.0.1:9000/batch' },
    '/batch/**': { proxy: 'http://127.0.0.1:9000/batch/**' },
    '/swagger': { proxy: 'http://127.0.0.1:9000/swagger' },
    '/swagger/**': { proxy: 'http://127.0.0.1:9000/swagger/**' }
  },

  experimental: {
    // Inline payload in HTML, extract for client-side navigation only.
    payloadExtraction: 'client'
  },

  compatibilityDate: '2025-01-15',

  vite: {
    resolve: {
      dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom']
    },
    plugins: [
      codecovVitePlugin({
        enableBundleAnalysis: process.env.CODECOV_TOKEN !== undefined,
        bundleName: 'enduro-frontend',
        uploadToken: process.env.CODECOV_TOKEN
      })
    ]
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
