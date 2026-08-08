<template>
  <div v-if="alerts.length" class="container-fluid">
    <div v-for="(alert, i) in alerts" :key="i" class="alert alert-dismissible fade show" :class="'alert-' + (alert.type || 'info')">
      {{ alert.message }}
      <button class="btn-close" @click="$emit('dismiss', i)"></button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ alerts: any[] }>()
defineEmits(['dismiss'])
</script>
