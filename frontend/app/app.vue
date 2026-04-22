<script setup>
const runtimeConfig = useRuntimeConfig()
const appBaseUrl = String(runtimeConfig.app.baseURL || '/')
const normalizedAppBaseUrl = appBaseUrl.endsWith('/') ? appBaseUrl : `${appBaseUrl}/`
const appFaviconSvgHref = `${normalizedAppBaseUrl}favicon.svg`

useHead({
  meta: [
    { name: 'viewport', content: 'width=device-width, initial-scale=1' }
  ],
  link: [
    { rel: 'icon', type: 'image/svg+xml', href: appFaviconSvgHref }
  ],
  htmlAttrs: {
    lang: 'en'
  }
})

const title = 'Enduro'
const description = 'A tool to support ingest and automation in digital preservation workflows.'

useSeoMeta({
  title,
  description,
  ogTitle: title,
  ogDescription: description
})
</script>

<template>
  <UApp>
    <div class="min-h-svh flex flex-col">
      <AppFrontendChangeBanner />
      <AppHeader />

      <main class="flex-1 min-h-0">
        <NuxtPage />
      </main>

      <AppFooter />
    </div>
  </UApp>
</template>
