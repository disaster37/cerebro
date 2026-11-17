<template>
  <div id="app" class="min-h-screen bg-neutral-50 text-neutral-900 transition-colors dark:bg-neutral-950 dark:text-neutral-100">
    <NavbarComponent />
    <AlertComponent :alerts="alerts" @dismiss="dismissAlert" />
    <main class="mx-auto max-w-7xl px-4 py-6">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'

const alerts = useState<any[]>('alerts', () => [])
const route = useRoute()

// Clear stale global alerts when navigating between routes so an error from a
// previous page never leaks into the next one.
watch(() => route.path, () => {
  alerts.value = []
})

function dismissAlert(index: number) {
  alerts.value.splice(index, 1)
}
</script>
