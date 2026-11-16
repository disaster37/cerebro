<template>
  <div>
    <h4>REST Console</h4>
    <div class="mt-3 grid grid-cols-1 gap-4 md:grid-cols-4">
      <div>
        <UButton size="sm" color="neutral" variant="outline" @click="loadIndices">Load Indices</UButton>
      </div>
      <div>
        <UFormField label="Method">
          <USelect v-model="method" :items="['GET', 'POST', 'PUT', 'DELETE']" class="w-full" />
        </UFormField>
      </div>
      <div>
        <UFormField label="Path">
          <UInput v-model="path" placeholder="_cat/indices" class="w-full" />
        </UFormField>
      </div>
      <div>
        <UFormField label="&nbsp;">
          <UButton block @click="execute">Execute</UButton>
        </UFormField>
      </div>
    </div>
    <div class="mt-3">
      <UFormField label="Body">
        <UTextarea v-model="body" :rows="5" class="w-full font-mono" />
      </UFormField>
    </div>
    <div v-if="result !== null" class="mt-3">
      <label class="text-sm font-medium">Response ({{ resultStatus }})</label>
      <pre class="overflow-x-auto rounded-lg border bg-neutral-100 p-4"><code>{{ JSON.stringify(result, null, 2) }}</code></pre>
    </div>
    <div v-if="history.length" class="mt-3">
      <h6>History</h6>
      <UTable :data="history" :columns="historyColumns" @select="(__, row) => selectHistory(row.original)" />
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const method = ref('GET')
const path = ref('_cat/indices')
const body = ref('')
const result = ref<any>(null)
const resultStatus = ref(0)
const history = ref<any[]>([])

const historyColumns = [
  { accessorKey: 'created_at', header: 'Time' },
  { accessorKey: 'method', header: 'Method' },
  { accessorKey: 'path', header: 'Path' },
  { accessorKey: 'body', header: 'Body', meta: { class: { td: 'max-w-[200px] truncate' } } },
]

onMounted(() => loadHistory())

async function loadIndices() {
  await apiCall('POST', '/rest', { host: clusterName.value })
}

async function execute() {
  const resp = await apiCall('POST', '/rest/request', {
    host: clusterName.value,
    method: method.value,
    path: path.value,
    data: body.value || undefined,
  })
  if (resp) {
    result.value = resp.body
    resultStatus.value = resp.status
    await loadHistory()
  }
}

async function loadHistory() {
  const resp = await apiCall('POST', '/rest/history', { host: clusterName.value })
  if (resp?.status === 200) history.value = resp.body || []
}

function selectHistory(h: any) {
  method.value = h.method
  path.value = h.path
  body.value = h.body
}
</script>
