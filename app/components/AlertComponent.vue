<template>
  <div v-if="alerts.length" class="mb-4 space-y-3">
    <UAlert
      v-for="(alert, i) in alerts"
      :key="i"
      :color="toColor(alert.type)"
      variant="soft"
      :description="alert.message"
      :close="true"
      @update:open="(open) => { if (!open) emit('dismiss', i) }"
    />
  </div>
</template>

<script setup lang="ts">
defineProps<{ alerts: any[] }>()
const emit = defineEmits<{ (e: 'dismiss', index: number): void }>()

function toColor(type?: string) {
  switch (type) {
    case 'danger': return 'error'
    case 'success': return 'success'
    case 'warning': return 'warning'
    case 'info': return 'info'
    default: return 'info'
  }
}
</script>
