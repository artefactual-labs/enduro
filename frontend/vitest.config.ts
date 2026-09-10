import { defineConfig } from 'vitest/config'
import { defineVitestProject } from '@nuxt/test-utils/config'

const nuxtProject = await defineVitestProject({
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

// Nuxt's production manifest plugin runs on test teardown, but Vitest never
// emits a production client manifest. Remove this workaround once
// @nuxt/test-utils excludes the plugin itself.
nuxtProject.plugins = nuxtProject.plugins?.filter(plugin =>
  !plugin || !('name' in plugin) || plugin.name !== 'nuxt:client-manifest'
)

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
      nuxtProject
    ]
  }
})
