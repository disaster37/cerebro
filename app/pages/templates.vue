<template>
  <div>
    <h4>Templates</h4>
    <UTable :data="templates" :columns="templatesColumns">
      <template #actions-cell="{ row }">
        <UButton size="sm" color="error" variant="soft" @click="del(row.original.name)">Delete</UButton>
      </template>
    </UTable>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const templates = ref<any[]>([])

const templatesColumns = [
  { accessorKey: 'name', header: 'Name' },
  { id: 'patterns', header: 'Patterns', cell: ({ row }: any) => row.original.template?.index_patterns?.join(', ') },
  { id: 'order', header: 'Order', cell: ({ row }: any) => row.original.template?.order },
  { id: 'actions', header: 'Actions' },
]

onMounted(async () => {
  const resp = await apiCall('POST', '/templates', { host: clusterHost.value })
  if (resp?.status === 200) templates.value = resp.body || []
})

async function del(name: string) {
  await apiCall('POST', '/templates/delete', { host: clusterHost.value, name })
  const resp = await apiCall('POST', '/templates', { host: clusterHost.value })
  if (resp?.status === 200) templates.value = resp.body || []
}
</script>
