<template>
  <div>
    <h4 class="mb-4 text-lg font-semibold">Create Index</h4>
    <form class="mt-3 max-w-3xl" @submit.prevent="create">
      <div class="mb-3">
        <UFormField label="Index Name">
          <UInput v-model="name" required class="w-full" data-testid="create-index-name" />
        </UFormField>
      </div>
      <div class="mb-3">
        <UFormField label="Settings / Mappings (JSON)">
          <UTextarea v-model="meta" :rows="12" class="w-full font-mono" data-testid="create-index-json" />
        </UFormField>
      </div>
      <UAlert v-if="jsonError" color="error" variant="soft" class="mb-3" :description="jsonError" data-testid="create-index-json-error" />
      <UAlert v-if="msg" :color="success ? 'success' : 'error'" variant="soft" class="mb-3" :description="msg" data-testid="create-index-result" />
      <UButton type="submit" :loading="creating" data-testid="create-index-submit">Create</UButton>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const name = ref('')
const meta = ref('{\n  "settings": {\n    "number_of_shards": 1\n  }\n}')
const msg = ref('')
const success = ref(false)
const jsonError = ref('')
const creating = ref(false)

function validateJson(): any | null {
  jsonError.value = ''
  if (!meta.value.trim()) return {}
  try {
    const parsed = JSON.parse(meta.value)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      jsonError.value = 'Settings / mappings must be a JSON object.'
      return null
    }
    return parsed
  } catch (err: any) {
    jsonError.value = `Invalid JSON: ${err.message}`
    return null
  }
}

async function create() {
  msg.value = ''
  success.value = false
  const metadata = validateJson()
  if (metadata === null) return

  creating.value = true
  const resp = await apiCall('POST', '/create_index/create', {
    host: clusterHost.value,
    index: name.value,
    metadata,
  }, { suppressErrorAlert: true })
  creating.value = false

  if (resp?.status === 200) {
    msg.value = 'Index created'
    success.value = true
  } else {
    msg.value = extractEnvelopeError(resp?.body) || `Failed to create index (${resp?.status || 'network error'}).`
    success.value = false
  }
}
</script>
