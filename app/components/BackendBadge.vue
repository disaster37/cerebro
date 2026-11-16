<template>
  <UBadge
    v-if="backendType"
    :color="backendType === 'opensearch' ? 'secondary' : 'primary'"
    variant="solid"
    data-testid="backend-badge"
  >
    {{ label }}
  </UBadge>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ backendType?: string; backendVersion?: string }>()

const label = computed(() => {
  const name = props.backendType === 'opensearch' ? 'OpenSearch' : 'Elasticsearch'
  return props.backendVersion ? `${name} ${props.backendVersion}` : name
})
</script>
