<template>
  <div>
    <h4>REST Console</h4>
    <div class="row mt-3">
      <div class="col-md-3">
        <button class="btn btn-sm btn-outline-secondary" @click="loadIndices">Load Indices</button>
      </div>
      <div class="col-md-3">
        <label class="form-label">Method</label>
        <select v-model="method" class="form-select">
          <option>GET</option><option>POST</option><option>PUT</option><option>DELETE</option>
        </select>
      </div>
      <div class="col-md-3">
        <label class="form-label">Path</label>
        <input v-model="path" class="form-control" placeholder="_cat/indices" />
      </div>
      <div class="col-md-3">
        <label class="form-label">&nbsp;</label>
        <button class="btn btn-primary w-100" @click="execute">Execute</button>
      </div>
    </div>
    <div class="mt-3">
      <label class="form-label">Body</label>
      <textarea v-model="body" class="form-control" rows="5" style="font-family:monospace"></textarea>
    </div>
    <div class="mt-3" v-if="result !== null">
      <label class="form-label">Response ({{ resultStatus }})</label>
      <pre class="bg-light p-3 border rounded"><code>{{ JSON.stringify(result, null, 2) }}</code></pre>
    </div>
    <div class="mt-3" v-if="history.length">
      <h6>History</h6>
      <table class="table table-sm">
        <thead><tr><th>Time</th><th>Method</th><th>Path</th><th>Body</th></tr></thead>
        <tbody>
          <tr v-for="(h, i) in history" :key="i" style="cursor:pointer" @click="selectHistory(h)">
            <td>{{ h.created_at }}</td><td>{{ h.method }}</td><td>{{ h.path }}</td><td class="text-truncate" style="max-width:200px">{{ h.body }}</td>
          </tr>
        </tbody>
      </table>
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

onMounted(() => loadHistory())

async function loadIndices() {
  const resp = await apiCall('POST', '/rest', { host: clusterName.value })
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
