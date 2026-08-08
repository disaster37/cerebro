<template>
  <div>
    <h4>Index Settings</h4>
    <div class="row">
      <div class="col-md-4">
        <label>Index</label>
        <input v-model="index" class="form-control" placeholder="my-index" />
        <button class="btn btn-sm btn-outline-secondary mt-2" @click="load">Load Settings</button>
      </div>
    </div>
    <div v-if="idxSettings" class="mt-3">
      <div v-for="(val, key) in idxSettings" :key="key" class="mb-2">
        <label>{{ key }}</label>
        <input :value="val" class="form-control form-control-sm" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const index = ref('')
const idxSettings = ref<any>(null)

async function load() {
  const resp = await apiCall('POST', '/index_settings', { host: clusterName.value, index: index.value })
  if (resp?.status === 200) idxSettings.value = resp.body
}
</script>
