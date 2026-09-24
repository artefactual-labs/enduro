import { createSharedComposable, syncRef, useLocalStorage } from '@vueuse/core'

export const useDisplaySettings = createSharedComposable(() => {
  const showConnectionMonitor = useLocalStorage('enduro-show-connection-monitor', true)
  // Reuse Nuxt's key so existing preferences and the initial page theme agree.
  const theme = useLocalStorage('nuxt-color-mode', 'system')
  const colorMode = useColorMode()

  // Let Nuxt apply the theme while VueUse also listens for storage changes.
  syncRef(theme, toRef(colorMode, 'preference'), {
    transform: {
      ltr: value => value === 'light' || value === 'dark' ? value : 'system'
    }
  })

  return { showConnectionMonitor }
})
