<template>
  <div>
    <h4>Connect to Elasticsearch / OpenSearch Cluster</h4>
    <form @submit.prevent="doConnect" class="mt-3 grid max-w-2xl gap-4">
      <UFormField label="Host">
        <USelect v-model="selectedHost" :items="hosts" placeholder="Select a host..." class="w-full" />
      </UFormField>
      <UFormField label="Custom Host URL">
        <UInput v-model="customHost" placeholder="http://localhost:9200" class="w-full" />
      </UFormField>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <UFormField label="Username">
          <UInput v-model="username" class="w-full" />
        </UFormField>
        <UFormField label="Password">
          <UInput v-model="password" type="password" class="w-full" />
        </UFormField>
      </div>
      <div>
        <UButton type="submit" :loading="connecting">Connect</UButton>
        <span v-if="error" class="ms-2 text-error-500">{{ error }}</span>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
const { hosts, load } = useHosts()
const { connect, loadNavbar } = useCluster()
const selectedHost = ref('')
const customHost = ref('')
const username = ref('')
const password = ref('')
const connecting = ref(false)
const error = ref('')

onMounted(() => load())

async function doConnect() {
  const host = customHost.value || selectedHost.value
  if (!host) return

  connecting.value = true
  error.value = ''
  const ok = await connect(host, username.value || undefined, password.value || undefined)
  if (ok) {
    await loadNavbar(host)
    navigateTo('/overview')
  } else {
    error.value = 'Connection failed'
  }
  connecting.value = false
}
</script>
