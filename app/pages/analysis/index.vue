<template>
  <div>
    <h4>Analysis</h4>
    <div class="row mt-3">
      <div class="col-md-4">
        <label>Index</label>
        <select v-model="selectedIndex" class="form-select" @change="onIndexChange">
          <option value="">Select...</option>
          <option v-for="idx in indices" :key="idx" :value="idx">{{ idx }}</option>
        </select>
      </div>
      <div class="col-md-4">
        <label>Field</label>
        <select v-model="selectedField" class="form-select">
          <option value="">Select...</option>
          <option v-for="f in fields" :key="f" :value="f">{{ f }}</option>
        </select>
      </div>
      <div class="col-md-4">
        <label>Text</label>
        <input v-model="text" class="form-control" />
      </div>
    </div>
    <button class="btn btn-primary mt-3" @click="analyze">Analyze</button>
    <div v-if="tokens" class="mt-3">
      <table class="table table-sm">
        <thead><tr><th>Token</th><th>Position</th><th>Type</th></tr></thead>
        <tbody>
          <tr v-for="t in tokens" :key="t.token">
            <td>{{ t.token }}</td><td>{{ t.position }}</td><td>{{ t.type }}</td>
          </tr>
        </tbody>
      </table>
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
