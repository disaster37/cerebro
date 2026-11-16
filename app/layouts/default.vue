<template>
  <div id="app" class="min-h-screen bg-neutral-50">
    <NavbarComponent />
    <AlertComponent :alerts="alerts" @dismiss="dismissAlert" />
    <main class="mx-auto max-w-7xl px-4 py-6">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
const alerts = useState<any[]>('alerts', () => [])
function dismissAlert(index: number) {
  alerts.value.splice(index, 1)
}
</script>
