<template>
  <div>
    <h4>Cluster Settings</h4>
    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <div v-else-if="settings" class="mt-3">
      <div v-for="(val, key) in flatSettings" :key="key" class="mb-2">
        <UFormField :label="String(key)">
          <UInput v-model="flatSettings[key]" size="sm" class="w-full" />
        </UFormField>
      </div>
      <UButton class="mt-2" @click="save">Save</UButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'

const { clusterHost } = useCluster()
const { apiCall } = useAPI()
const settings = ref<any>(null)
const flatSettings = ref<any>({})
const loading = ref(true)

onMounted(async () => {
  const resp = await apiCall('POST', '/cluster_settings', { host: clusterHost.value })
  if (resp?.status === 200) {
    settings.value = resp.body
    const f: any = {}
    if (resp.body?.transient) Object.assign(f, resp.body.transient)
    if (resp.body?.persistent) Object.assign(f, resp.body.persistent)
    flatSettings.value = f
  }
  loading.value = false
})

async function save() {
  await apiCall('POST', '/cluster_settings/save', {
    host: clusterHost.value,
    settings: { transient: flatSettings.value },
  })
}
</script>
