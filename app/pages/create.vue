<template>
  <div>
    <h4>Create Index</h4>
    <form @submit.prevent="create" class="mt-3">
      <div class="mb-3">
        <UFormField label="Index Name">
          <UInput v-model="name" required class="w-full" />
        </UFormField>
      </div>
      <div class="mb-3">
        <UFormField label="Settings / Mappings (JSON)">
          <UTextarea v-model="meta" :rows="10" class="w-full font-mono" />
        </UFormField>
      </div>
      <UButton type="submit">Create</UButton>
      <span v-if="msg" class="ms-2" :class="success ? 'text-success-500' : 'text-error-500'">{{ msg }}</span>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const name = ref('')
const meta = ref('{}')
const msg = ref('')
const success = ref(false)

async function create() {
  const resp = await apiCall('POST', '/create_index/create', {
    host: clusterHost.value,
    index: name.value,
    metadata: meta.value ? JSON.parse(meta.value) : {},
  })
  if (resp?.status === 200) { msg.value = 'Index created'; success.value = true }
  else { msg.value = 'Failed'; success.value = false }
}
</script>
