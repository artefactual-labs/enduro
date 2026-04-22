<script setup lang="ts">
const runtimeConfig = useRuntimeConfig()
const versionLabel = useState<string>('enduroVersion', () => '')
const appBaseUrl = String(runtimeConfig.app.baseURL || '/')
const normalizedAppBaseUrl = appBaseUrl.endsWith('/') ? appBaseUrl : `${appBaseUrl}/`
const footerAvatarSrc = `${normalizedAppBaseUrl}favicon-artefactual.ico`
</script>

<template>
  <USeparator
    :avatar="{
      src: footerAvatarSrc
    }"
  />
  <UFooter :ui="{ container: 'lg:items-stretch', center: 'lg:items-start' }">
    <template #left>
      <p class="text-xs text-muted leading-relaxed">
        Published under the <a href="https://www.apache.org/licenses/LICENSE-2.0">Apache License 2.0</a>.<br>
        © {{ new Date().getFullYear() }} Artefactual Systems, Inc.
      </p>
    </template>

    <template
      v-if="versionLabel"
      #default
    >
      <p class="text-xs text-muted">
        {{ versionLabel }}
      </p>
    </template>

    <template #right>
      <div class="flex items-center gap-1">
        <AppConnectionMonitor />
        <UButton
          to="https://github.com/artefactual-labs/enduro"
          target="_blank"
          rel="noopener noreferrer"
          icon="i-simple-icons-github"
          aria-label="GitHub"
          color="neutral"
          variant="ghost"
        />
      </div>
    </template>
  </UFooter>
</template>
