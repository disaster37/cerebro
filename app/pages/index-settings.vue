<template>
  <div>
    <h4 class="mb-4 text-lg font-semibold">Index Settings</h4>

    <div class="mb-4 flex flex-wrap items-end gap-3">
      <UFormField label="Index">
        <UInput v-model="index" placeholder="my-index" class="w-64" data-testid="index-settings-index" />
      </UFormField>
      <UButton size="sm" color="neutral" variant="outline" :loading="loading" @click="load">Load Settings</UButton>
      <UButton v-if="Object.keys(draft).length" size="sm" :loading="saving" data-testid="index-settings-save" @click="save">
        Save Changes ({{ Object.keys(draft).length }})
      </UButton>
    </div>

    <UAlert v-if="error" color="error" variant="soft" class="mb-3" :description="error" />
    <UAlert v-if="success" color="success" variant="soft" class="mb-3" description="Index settings saved." />

    <div v-if="settings" class="space-y-3">
      <UInput v-model="filter" size="sm" icon="i-lucide-search" placeholder="Filter settings..." class="w-64" />
      <div class="max-h-[60vh] space-y-2 overflow-y-auto">
        <div v-for="key in filteredKeys" :key="key" class="grid grid-cols-1 items-center gap-2 md:grid-cols-[1fr_20rem]">
          <code class="text-xs">{{ key }}</code>
          <UInput
            :model-value="draft[key] ?? settings[key]"
            size="sm"
            class="w-full"
            @update:model-value="draft[key] = $event"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const route = useRoute()

const index = ref('')
const settings = ref<Record<string, any> | null>(null)
const draft = ref<Record<string, any>>({})
const filter = ref('')
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref(false)

const filteredKeys = computed(() => {
  const keys = Object.keys(settings.value || {})
  if (!filter.value) return keys
  return keys.filter((key) => key.toLowerCase().includes(filter.value.toLowerCase()))
})

function normalizeSettings(body: any, indexName: string): Record<string, any> {
  if (body && typeof body === 'object' && body[indexName]?.settings) {
    return body[indexName].settings
  }
  if (body && typeof body === 'object' && body.settings) {
    return body.settings
  }
  return body || {}
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    return
  }
  if (!index.value) {
    error.value = 'Index name is required.'
    return
  }
  loading.value = true
  error.value = ''
  success.value = false
  const resp = await apiCall('POST', '/index_settings', { host: clusterHost.value, index: index.value }, { suppressErrorAlert: true })
  loading.value = false
  if (resp?.status === 200) {
    settings.value = normalizeSettings(resp.body, index.value)
    draft.value = {}
  } else {
    error.value = extractEnvelopeError(resp?.body) || `Failed to load index settings (${resp?.status || 'network error'}).`
  }
}

async function save() {
  saving.value = true
  error.value = ''
  success.value = false
  const resp = await apiCall('POST', '/index_settings/update', {
    host: clusterHost.value,
    index: index.value,
    settings: draft.value,
  }, { suppressErrorAlert: true })
  saving.value = false
  if (resp?.status === 200) {
    await load()
    success.value = true
  } else {
    error.value = extractEnvelopeError(resp?.body) || `Failed to save index settings (${resp?.status || 'network error'}).`
  }
}

onMounted(() => {
  const queryIndex = route?.query?.index
  if (typeof queryIndex === 'string' && queryIndex) {
    index.value = queryIndex
    load()
  }
})
</script>
