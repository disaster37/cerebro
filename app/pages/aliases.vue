<template>
  <div>
    <h4>Aliases</h4>
    <div class="row">
      <div class="col-md-6">
        <table class="table table-sm">
          <thead><tr><th>Alias</th><th>Index</th><th>Filter</th></tr></thead>
          <tbody>
            <tr v-for="(a, i) in aliases" :key="i">
              <td>{{ a.alias }}</td><td>{{ a.index }}</td>
              <td><small><code>{{ a.filter }}</code></small></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const aliases = ref<any[]>([])

onMounted(async () => {
  const resp = await apiCall('POST', '/aliases/get_aliases', { host: clusterName.value })
  if (resp?.status === 200) aliases.value = resp.body || []
})
</script>
