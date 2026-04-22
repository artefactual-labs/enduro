import { createEnduroApiClient, resolveEnduroApiBasePath } from '~/utils/enduro-api-client'

export default defineNuxtPlugin({
  name: 'enduro-api',
  setup() {
    const runtimeConfig = useRuntimeConfig()
    const basePath = resolveEnduroApiBasePath(runtimeConfig)
    const enduroApi = createEnduroApiClient(basePath)

    return {
      provide: {
        enduroApi
      }
    }
  }
})
