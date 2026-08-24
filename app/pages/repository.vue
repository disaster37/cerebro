<template>
  <div>
    <h4>Repositories</h4>
    <table class="table table-sm"><thead><tr><th>Name</th><th>Type</th></tr></thead>
    <tbody><tr v-for="r in repos" :key="r.name"><td>{{ r.name }}</td><td>{{ r.type }}</td></tr></tbody></table>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const repos = ref<any[]>([])

onMounted(async () => {
  const resp = await apiCall('POST', '/repositories', { host: clusterName.value })
  if (resp?.status === 200) repos.value = resp.body || []
})
</script>
