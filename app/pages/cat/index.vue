<template>
  <div>
    <h4>Cat API</h4>
    <div class="row">
      <div class="col-md-4">
        <input v-model="api" class="form-control" placeholder="indices" />
      </div>
      <div class="col-md-2">
        <button class="btn btn-primary" @click="execute">Execute</button>
      </div>
    </div>
    <pre v-if="result" class="mt-3 bg-light p-3 border rounded"><code>{{ JSON.stringify(result, null, 2) }}</code></pre>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const api = ref('indices')
const result = ref<any>(null)

async function execute() {
  const resp = await apiCall('POST', '/cat', { host: clusterName.value, api: api.value })
  if (resp?.status === 200) result.value = resp.body
}
</script>
