<template>
  <div>
    <h4>Create Index</h4>
    <form @submit.prevent="create" class="mt-3">
      <div class="mb-3"><label>Index Name</label><input v-model="name" class="form-control" required /></div>
      <div class="mb-3"><label>Settings / Mappings (JSON)</label><textarea v-model="meta" class="form-control" rows="10" style="font-family:monospace"></textarea></div>
      <button class="btn btn-primary">Create</button>
      <span v-if="msg" class="ms-2" :class="{ 'text-success': success, 'text-danger': !success }">{{ msg }}</span>
    </form>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const name = ref('')
const meta = ref('{}')
const msg = ref('')
const success = ref(false)

async function create() {
  const resp = await apiCall('POST', '/create_index/create', {
    host: clusterName.value,
    index: name.value,
    metadata: meta.value ? JSON.parse(meta.value) : {},
  })
  if (resp?.status === 200) { msg.value = 'Index created'; success.value = true }
  else { msg.value = 'Failed'; success.value = false }
}
</script>
