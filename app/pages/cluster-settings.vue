<template>
  <div>
    <h4>Cluster Settings</h4>
    <div v-if="settings" class="mt-3">
      <div v-for="(val, key) in flatSettings" :key="key" class="mb-2">
        <UFormField :label="key">
          <UInput v-model="flatSettings[key]" size="sm" class="w-full" />
        </UFormField>
      </div>
      <UButton class="mt-2" @click="save">Save</UButton>
    </div>
    <div v-else class="flex justify-center p-5">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
  </div>
</template>

<script setup lang="ts">
const clusterName = useState<string>('clusterName')
const { apiCall } = useAPI()
const settings = ref<any>(null)
const flatSettings = ref<any>({})

onMounted(async () => {
  const resp = await apiCall('POST', '/cluster_settings', { host: clusterName.value })
  if (resp?.status === 200) {
    settings.value = resp.body
    // flatten transient/persistent
    const f: any = {}
    if (resp.body?.transient) Object.assign(f, resp.body.transient)
    if (resp.body?.persistent) Object.assign(f, resp.body.persistent)
    flatSettings.value = f
  }
})

async function save() {
  await apiCall('POST', '/cluster_settings/save', {
    host: clusterName.value,
    settings: { transient: flatSettings.value },
  })
}
</script>
