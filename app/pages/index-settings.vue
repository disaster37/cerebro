<template>
  <div>
    <h4>Index Settings</h4>
    <div class="max-w-sm">
      <UFormField label="Index">
        <UInput v-model="index" placeholder="my-index" class="w-full" />
      </UFormField>
      <UButton size="sm" color="neutral" variant="outline" class="mt-2" @click="load">Load Settings</UButton>
    </div>
    <div v-if="idxSettings" class="mt-3">
      <div v-for="(val, key) in idxSettings" :key="key" class="mb-2">
        <UFormField :label="key">
          <UInput :value="val" size="sm" readonly class="w-full" />
        </UFormField>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const index = ref('')
const idxSettings = ref<any>(null)

async function load() {
  const resp = await apiCall('POST', '/index_settings', { host: clusterName.value, index: index.value })
  if (resp?.status === 200) idxSettings.value = resp.body
}
</script>
