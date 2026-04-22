import type { EnduroApiClient } from '~/utils/enduro-api-client'

declare module '#app' {
  interface NuxtApp {
    $enduroApi: EnduroApiClient
  }
}

declare module 'vue' {
  interface ComponentCustomProperties {
    $enduroApi: EnduroApiClient
  }
}

export {}
