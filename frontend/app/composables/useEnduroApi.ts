import type { EnduroApiClient } from '~/utils/enduro-api-client'

export function useEnduroApi(): EnduroApiClient {
  return useNuxtApp().$enduroApi
}
