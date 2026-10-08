<template>
  <div>
    <h4>Aliases</h4>
    <div class="max-w-3xl">
      <UTable :data="aliases" :columns="aliasesColumns">
        <template #filter-cell="{ row }">
          <code class="text-xs">{{ row.original.filter }}</code>
        </template>
      </UTable>
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const aliases = ref<any[]>([])

const aliasesColumns = [
  { accessorKey: 'alias', header: 'Alias' },
  { accessorKey: 'index', header: 'Index' },
  { id: 'filter', header: 'Filter' },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/aliases/get_aliases', { host: clusterName.value })
  if (resp?.status === 200) aliases.value = resp.body || []
})
</script>
