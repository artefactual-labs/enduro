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

  future: {
    compatibilityVersion: 5
  },

  experimental: {
    // Inline payload in HTML, extract for client-side navigation only.
    payloadExtraction: 'client',
    // Preserve named page exports so Vue Router can discover and rerun data
    // loaders during client-side navigation.
    // TODO: Remove once Nuxt's normalized page-name wrapper preserves the
    // named exports required by Vue Router's experimental data loaders.
    normalizePageNames: false,
    // Enable Nitro auto-imports so Nuxt Icon and generated app configuration
    // imports are transformed correctly in compatibility mode.
    // TODO: Remove once https://github.com/nuxt/icon/issues/467 and
    // https://github.com/nuxt/nuxt/issues/34142 are fixed.
    nitroAutoImports: true
  },

  compatibilityDate: '2025-01-15',

  vite: {
    resolve: {
      dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom']
    }
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
