<template>
  <div class="py-6 sm:py-10">
    <UCard
      class="connect-card mx-auto max-w-md"
      :ui="{
        root: 'overflow-hidden bg-white shadow-sm ring-1 ring-neutral-200',
        header: 'bg-neutral-900 px-6 py-4 text-white',
        body: 'bg-white p-6',
      }"
    >
      <template #header>
        <h4 class="text-base font-semibold text-white">Connect to Cluster</h4>
        <p class="mt-0.5 text-xs text-neutral-300">Elasticsearch or OpenSearch endpoint</p>
      </template>

      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        :description="error"
        role="alert"
        data-testid="connect-error"
        class="mb-4"
      />

      <form class="space-y-4" novalidate @submit.prevent="doConnect">
        <UFormField label="Configured Host">
          <USelect
            v-model="selectedHost"
            :items="hosts"
            placeholder="Select a host..."
            class="w-full"
            data-testid="connect-host"
            @update:model-value="onHostChange"
          />
        </UFormField>

        <UFormField label="Custom Host URL">
          <UInput
            v-model="customHost"
            type="url"
            placeholder="https://localhost:9200"
            autocomplete="url"
            class="w-full"
            data-testid="connect-custom-host"
          />
        </UFormField>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <UFormField label="Username">
            <UInput
              v-model="username"
              type="text"
              placeholder="Optional"
              autocomplete="username"
              class="w-full"
              data-testid="connect-username"
            />
          </UFormField>
          <UFormField label="Password">
            <UInput
              v-model="password"
              type="password"
              placeholder="Optional"
              autocomplete="current-password"
              class="w-full"
              data-testid="connect-password"
            />
          </UFormField>
        </div>

        <div class="rounded-lg border border-neutral-200 bg-neutral-50 p-3">
          <UCheckbox
            v-model="trustCert"
            label="Trust self-signed certificate"
            description="Warning: Disables TLS certificate verification for this cluster connection, exposing traffic to potential man-in-the-middle interception. Enable only for trusted self-signed or development clusters."
            data-testid="connect-trust-cert"
          />
        </div>

        <UButton
          type="submit"
          color="primary"
          block
          :loading="connecting"
          :disabled="connecting || !resolvedHost"
          data-testid="connect-submit"
        >
          {{ connecting ? 'Connecting...' : 'Connect' }}
        </UButton>
      </form>
    </UCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useCluster } from '../composables/useCluster'
import { useHosts } from '../composables/useHosts'

const { hosts, load } = useHosts()
const { connect, loadNavbar, lastConnectError } = useCluster()

const selectedHost = ref('')
const customHost = ref('')
const username = ref('')
const password = ref('')
const trustCert = ref(false)
const connecting = ref(false)
const error = ref('')

const resolvedHost = computed(() => customHost.value.trim() || selectedHost.value.trim())

onMounted(() => load())

function onHostChange(val?: string) {
  const next = typeof val === 'string' ? val : selectedHost.value
  if (next) {
    customHost.value = ''
  }
}

async function doConnect() {
  const host = resolvedHost.value
  if (!host) return

  connecting.value = true
  error.value = ''
  const ok = await connect(
    host,
    username.value || undefined,
    password.value || undefined,
    trustCert.value,
  )
  if (ok) {
    await loadNavbar(host)
    navigateTo('/overview')
  } else if (!trustCert.value && /x509:|certificate/i.test(lastConnectError.value || '')) {
    error.value = 'Connection failed: TLS certificate verification failed. Enable "Trust self-signed certificate" if this cluster uses a self-signed certificate.'
  } else {
    error.value = 'Connection failed'
  }
  connecting.value = false
}
</script>
