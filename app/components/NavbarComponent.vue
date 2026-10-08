<template>
  <header class="sticky top-0 z-50 border-b border-neutral-200 bg-white">
    <div class="mx-auto flex max-w-7xl items-center gap-4 px-4 py-2">
      <NuxtLink to="/" class="text-sm font-semibold">cerebro</NuxtLink>

      <UNavigationMenu
        :items="navItems"
        orientation="horizontal"
        class="hidden flex-1 md:flex"
      >
        <template #connect-label="{ item }">
          <span class="flex items-center gap-2">{{ item.label }}
            <BackendBadge :backend-type="backendType" :backend-version="backendVersion" />
          </span>
        </template>
      </UNavigationMenu>

      <!-- Mobile menu (hamburger) -->
      <UDropdownMenu
        class="md:hidden"
        :items="mobileItems"
        :content="{ align: 'end' }"
      >
        <UButton icon="i-lucide-menu" color="neutral" variant="ghost" aria-label="Menu" />
      </UDropdownMenu>

      <div class="ml-auto flex items-center gap-2">
        <span v-if="user" class="text-sm text-neutral-500">{{ user }}</span>
        <UButton v-if="!user" to="/login" variant="ghost" color="neutral">Login</UButton>
        <UButton v-else variant="ghost" color="neutral" @click="logout">Logout</UButton>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const clusterName = useState<string>('clusterName', () => '')
const user = useState<string>('user', () => '')
const backendType = useState<string>('backendType', () => '')
const backendVersion = useState<string>('backendVersion', () => '')

const connectLabel = computed(() => clusterName.value || 'Connect')

const primaryLinks = [
  { label: 'Overview', to: '/overview' },
  { label: 'Nodes', to: '/nodes' },
  { label: 'REST', to: '/rest' },
  { label: 'Aliases', to: '/aliases' },
  { label: 'Create Index', to: '/create' },
  { label: 'Analysis', to: '/analysis' },
  { label: 'Templates', to: '/templates' },
]

const moreLinks = [
  { label: 'Cluster Settings', to: '/cluster-settings' },
  { label: 'Index Settings', to: '/index-settings' },
  { label: 'Snapshot', to: '/snapshot' },
  { label: 'Repositories', to: '/repository' },
  { label: 'CAT', to: '/cat' },
]

// Horizontal nav (desktop). "More" renders its children as a dropdown.
const navItems = computed(() => {
  const connect = { label: 'Connect', to: '/connect', slot: 'connect' }
  if (!clusterName.value) return [connect]
  return [connect, ...primaryLinks, { label: 'More', children: moreLinks }]
})

// Mobile menu items (all links, flat).
const mobileItems = computed(() => {
  const connect = { label: connectLabel.value, to: '/connect' }
  if (!clusterName.value) return [connect]
  return [connect, ...primaryLinks, ...moreLinks]
})

async function logout() {
  await $fetch('/auth/logout', { method: 'POST' })
  navigateTo('/login')
}
</script>
