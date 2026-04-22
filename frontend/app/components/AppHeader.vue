<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const route = useRoute()

function createNavItem(item: Omit<NavigationMenuItem, 'active'> & { active: boolean }): NavigationMenuItem {
  return {
    ...item,
    class: item.active ? 'text-[var(--app-on-brand)] before:bg-[var(--app-brand-strong)]' : undefined,
    ui: item.active ? { linkLeadingIcon: 'text-[var(--app-on-brand)] group-data-[state=open]:text-[var(--app-on-brand)]' } : undefined
  }
}

const items = computed<NavigationMenuItem[]>(() => [
  createNavItem({
    label: 'Collections',
    to: '/collections',
    icon: 'i-lucide-list-collapse',
    active: route.path.startsWith('/collections')
  }),
  createNavItem({
    label: 'Pipelines',
    to: '/pipelines',
    icon: 'i-lucide-boxes',
    active: route.path.startsWith('/pipelines')
  })
])
</script>

<template>
  <UHeader class="border-b border-default bg-slate-50/90 dark:bg-slate-950/60">
    <template #title>
      <NuxtLink to="/collections">
        <AppLogo class="shrink-0" />
      </NuxtLink>
    </template>
    <UNavigationMenu
      :items="items"
    />
    <template #right>
      <UColorModeButton />
    </template>
    <template #body>
      <UNavigationMenu
        :items="items"
        orientation="vertical"
        class="-mx-2.5"
      />
    </template>
  </UHeader>
</template>
