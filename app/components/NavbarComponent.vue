<template>
  <header class="sticky top-0 z-50 border-b bg-white/95 backdrop-blur dark:bg-neutral-950/95" :class="healthBorder">
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
        <span
          v-if="clusterStatus"
          class="inline-flex items-center gap-1 text-xs text-neutral-500"
          :data-testid="`cluster-health-${clusterStatus}`"
        >
          <span class="size-2 rounded-full" :class="healthDot" />
          {{ clusterStatus }}
        </span>
        <UColorModeSwitch size="sm" data-testid="color-mode-toggle" />
        <span v-if="displayUser" class="text-sm text-neutral-500" data-testid="navbar-user">{{ displayUser }}</span>
        <UButton
          v-if="authEnabled && !user"
          to="/login"
          variant="ghost"
          color="neutral"
          data-testid="navbar-login"
        >
          Login
        </UButton>
        <UButton
          v-if="authEnabled && user"
          variant="ghost"
          color="neutral"
          data-testid="navbar-logout"
          @click="logout"
        >
          Logout
        </UButton>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import BackendBadge from './BackendBadge.vue'
import { defaultClusterConnection, type ClusterConnection } from '../composables/useAPI'
import { useAuth } from '../composables/useAuth'

const connection = useState<ClusterConnection>('clusterConnection', defaultClusterConnection)
const clusterName = useState<string>('clusterName', () => '')
const backendType = useState<string>('backendType', () => '')
const backendVersion = useState<string>('backendVersion', () => '')
const clusterStatus = useState<string>('clusterStatus', () => '')
const { user, authEnabled, checkSession, logout } = useAuth()

const connectLabel = computed(() => clusterName.value || 'Connect')
const displayUser = computed(() => connection.value.username || user.value)

const primaryLinks = [
  { label: 'Overview', to: '/overview' },
  { label: 'Indexes', to: '/indices' },
  { label: 'Nodes', to: '/nodes' },
  { label: 'DevTools', to: '/devtools' },
  { label: 'Policies', to: '/policies' },
  { label: 'Snapshots', to: '/snapshot' },
  { label: 'Repositories', to: '/repository' },
]

const moreLinks = [
  { label: 'Aliases', to: '/aliases' },
  { label: 'Templates', to: '/templates' },
  { label: 'Create Index', to: '/create' },
  { label: 'Cluster Settings', to: '/cluster-settings' },
  { label: 'Index Settings', to: '/index-settings' },
  { label: 'Analysis', to: '/analysis' },
  { label: 'CAT', to: '/cat' },
]

const healthBorder = computed(() => {
  switch (clusterStatus.value) {
    case 'green': return 'border-emerald-500'
    case 'yellow': return 'border-amber-500'
    case 'red': return 'border-rose-500'
    default: return 'border-neutral-200 dark:border-neutral-800'
  }
})

const healthDot = computed(() => {
  switch (clusterStatus.value) {
    case 'green': return 'bg-emerald-500'
    case 'yellow': return 'bg-amber-500'
    case 'red': return 'bg-rose-500'
    default: return 'bg-neutral-400'
  }
})

// Horizontal nav (desktop). "More" renders its children as a dropdown.
const navItems = computed(() => {
  const connect = { label: connectLabel.value, to: '/connect', slot: 'connect' }
  if (!clusterName.value) return [connect]
  return [connect, ...primaryLinks, { label: 'More', children: moreLinks }]
})

// Mobile menu items (all links, flat).
const mobileItems = computed(() => {
  const connect = { label: connectLabel.value, to: '/connect' }
  if (!clusterName.value) return [connect]
  return [connect, ...primaryLinks, ...moreLinks]
})

onMounted(() => {
  checkSession()
})
</script>
