<template>
  <div>
    <h4>Analysis</h4>
    <div class="mt-3 grid grid-cols-1 gap-4 md:grid-cols-3">
      <UFormField label="Index">
        <USelect v-model="selectedIndex" :items="indices" placeholder="Select..." class="w-full" @update:model-value="onIndexChange" />
      </UFormField>
      <UFormField label="Field">
        <USelect v-model="selectedField" :items="fields" placeholder="Select..." class="w-full" />
      </UFormField>
      <UFormField label="Text">
        <UInput v-model="text" class="w-full" />
      </UFormField>
    </div>
    <UButton class="mt-3" @click="analyze">Analyze</UButton>
    <div v-if="tokens" class="mt-3">
      <UTable :data="tokens" :columns="tokensColumns" />
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const indices = ref<string[]>([])
const fields = ref<string[]>([])
const selectedIndex = ref('')
const selectedField = ref('')
const text = ref('')
const tokens = ref<any[]>([])

const tokensColumns = [
  { accessorKey: 'token', header: 'Token' },
  { accessorKey: 'position', header: 'Position' },
  { accessorKey: 'type', header: 'Type' },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/analysis/indices', { host: clusterName.value })
  if (resp?.status === 200) indices.value = resp.body || []
})

async function onIndexChange() {
  if (!selectedIndex.value) return
  const resp = await apiCall('POST', '/analysis/fields', { host: clusterName.value, index: selectedIndex.value })
  if (resp?.status === 200) fields.value = resp.body || []
}

async function analyze() {
  const resp = await apiCall('POST', '/analysis/analyze/field', {
    host: clusterName.value, index: selectedIndex.value, field: selectedField.value, text: text.value,
  })
  if (resp?.status === 200) tokens.value = resp.body || []
}
</script>
