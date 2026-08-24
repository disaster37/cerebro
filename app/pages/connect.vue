<template>
  <div>
    <h4>Connect to Elasticsearch / OpenSearch Cluster</h4>
    <form @submit.prevent="doConnect" class="row g-3 mt-3" style="max-width:600px">
      <div class="col-12">
        <label class="form-label">Host</label>
        <select v-model="selectedHost" class="form-select" @change="onHostChange">
          <option value="">Select a host...</option>
          <option v-for="h in hosts" :key="h" :value="h">{{ h }}</option>
        </select>
      </div>
      <div class="col-12">
        <label class="form-label">Custom Host URL</label>
        <input v-model="customHost" class="form-control" placeholder="http://localhost:9200" />
      </div>
      <div class="col-6">
        <label class="form-label">Username</label>
        <input v-model="username" class="form-control" />
      </div>
      <div class="col-6">
        <label class="form-label">Password</label>
        <input v-model="password" type="password" class="form-control" />
      </div>
      <div class="col-12">
        <button class="btn btn-primary" :disabled="connecting">Connect</button>
        <span v-if="error" class="text-danger ms-2">{{ error }}</span>
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

function onHostChange() {
  if (selectedHost.value === 'custom') {
    selectedHost.value = ''
  }
}

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
