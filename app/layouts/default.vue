<template>
  <div id="app">
    <NavbarComponent />
    <AlertComponent :alerts="alerts" @dismiss="dismissAlert" />
    <div class="container-fluid mt-3">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
const alerts = useState<any[]>('alerts', () => [])
function dismissAlert(index: number) {
  alerts.value.splice(index, 1)
}
</script>
