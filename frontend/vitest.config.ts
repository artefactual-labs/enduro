import { defineConfig } from 'vitest/config'
import { defineVitestProject } from '@nuxt/test-utils/config'

export default defineConfig({
  resolve: {
    dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom']
  },
  test: {
    coverage: {
      enabled: false,
      provider: 'v8',
      reporter: ['text', 'lcov'],
      exclude: [
        'app/**/*.spec.ts',
        'app/**/*.nuxt.spec.ts',
        'app/openapi-generator/**',
        'app/types/**'
      ]
    },
    projects: [
      {
        test: {
          name: 'unit',
          include: ['app/**/*.spec.ts'],
          exclude: ['app/**/*.nuxt.spec.ts', 'app/openapi-generator/**'],
          environment: 'node'
        }
      },
      await defineVitestProject({
        test: {
          name: 'nuxt',
          include: ['app/**/*.nuxt.spec.ts'],
          setupFiles: ['vitest.setup.nuxt.ts'],
          environment: 'nuxt',
          environmentOptions: {
            nuxt: {
              domEnvironment: 'happy-dom',
              mock: {
                intersectionObserver: true
              }
            }
          }
        }
      })
    ]
  }
})
