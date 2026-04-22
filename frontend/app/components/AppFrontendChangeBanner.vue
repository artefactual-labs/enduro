<script setup lang="ts">
const open = ref(true)
const docsBase = 'https://github.com/artefactual-labs/enduro/blob/main/docs/configuration-reference.md'
const legacyListenHref = `${docsBase}#legacylisten-string`
</script>

<template>
  <UAlert
    v-if="open"
    close
    color="primary"
    variant="solid"
    orientation="horizontal"
    :ui="{
      root: 'rounded-none border-0 ring-0 shadow-none py-2.5',
      wrapper: 'min-w-0 text-center',
      description: 'text-xs sm:text-sm text-[var(--app-on-brand)] text-center',
      close: 'rounded-sm text-[var(--app-on-brand)] hover:text-[var(--app-on-brand)] hover:bg-[var(--app-on-brand-hover)]'
    }"
    @update:open="open = $event"
  >
    <template #description>
      Welcome to the experimental UI rewrite. The <ULink
        :to="legacyListenHref"
        target="_blank"
        rel="noopener noreferrer"
        class="font-semibold text-[var(--app-on-brand)] underline decoration-[var(--app-on-brand-muted)] underline-offset-2 hover:text-[var(--app-on-brand)] hover:decoration-[var(--app-on-brand)]"
      >current UI is still available</ULink> for now if you need it.
    </template>
  </UAlert>
</template>
