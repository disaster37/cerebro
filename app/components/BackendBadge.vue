<template>
  <span
    v-if="backendType"
    class="badge ms-2 align-middle"
    :class="badgeClass"
    data-testid="backend-badge"
  >{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ backendType?: string; backendVersion?: string }>()

const label = computed(() => {
  const name = props.backendType === 'opensearch' ? 'OpenSearch' : 'Elasticsearch'
  return props.backendVersion ? `${name} ${props.backendVersion}` : name
})

const badgeClass = computed(() =>
  props.backendType === 'opensearch' ? 'badge-opensearch' : 'text-bg-primary',
)
</script>
