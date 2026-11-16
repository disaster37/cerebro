<template>
  <div>
    <h4>Cat API</h4>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_auto]">
      <UInput v-model="api" placeholder="indices" class="w-full" />
      <UButton @click="execute">Execute</UButton>
    </div>
    <pre v-if="result" class="mt-3 overflow-x-auto rounded-lg border bg-neutral-100 p-4"><code>{{ JSON.stringify(result, null, 2) }}</code></pre>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const api = ref('indices')
const result = ref<any>(null)

async function execute() {
  const resp = await apiCall('POST', '/cat', { host: clusterHost.value, api: api.value })
  if (resp?.status === 200) result.value = resp.body
}
</script>
