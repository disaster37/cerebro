<template>
  <div>
    <h4>Snapshots</h4>
    <table class="table table-sm"><thead><tr><th>Snapshot</th><th>State</th><th>Indices</th></tr></thead>
    <tbody><tr v-for="s in snapshots" :key="s.snapshot"><td>{{ s.snapshot }}</td><td>{{ s.state }}</td><td>{{ s.indices?.join(', ') }}</td></tr></tbody></table>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const snapshots = ref<any[]>([])
const repos = ref<string[]>([])

onMounted(async () => {
  const resp = await apiCall('POST', '/snapshots', { host: clusterName.value })
  if (resp?.status === 200) {
    repos.value = resp.body?.repositories || []
    if (repos.value[0]) await loadSnaps(repos.value[0])
  }
})

async function loadSnaps(repo: string) {
  const resp = await apiCall('POST', '/snapshots/load', { host: clusterName.value, repository: repo })
  if (resp?.status === 200) snapshots.value = resp.body || []
}
</script>
