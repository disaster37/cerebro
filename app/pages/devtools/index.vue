<template>
  <div>
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">
        DevTools <small class="text-neutral-500">{{ clusterName }}</small>
      </h4>
      <div class="flex flex-wrap items-center gap-2">
        <UBadge v-if="activeRequest" color="info" variant="soft" data-testid="active-request">
          {{ activeRequest.method }} {{ activeRequest.path }}
        </UBadge>
        <UButton size="sm" icon="i-lucide-play" :loading="sending" @click="send">Send</UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-indent" @click="format">Auto-indent</UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-terminal" @click="copyCurl()">Copy as cURL</UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" @click="loadSchema">Reload Schema</UButton>
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-history" @click="historyOpen = true">History</UButton>
        <UButton size="sm" color="neutral" variant="ghost" icon="i-lucide-keyboard" @click="shortcutsOpen = true">Shortcuts</UButton>
      </div>
    </div>

    <UAlert
      v-if="copyFeedback"
      color="success"
      variant="soft"
      class="mb-3"
      data-testid="copy-curl-feedback"
      :description="copyFeedback"
    />

    <UAlert
      v-if="lintErrors.length"
      color="error"
      variant="soft"
      class="mb-3"
      data-testid="console-lint-banner"
    >
      <template #description>
        <div class="flex flex-wrap items-center gap-2">
          <span>{{ lintErrors.length }} syntax error(s) in the active request.</span>
          <UButton size="xs" color="error" variant="soft" @click="sendAnyway = true; send()">Send anyway</UButton>
        </div>
      </template>
    </UAlert>

    <div class="console-split" :class="{ 'console-split--stacked': stacked }">
      <div class="console-pane">
        <KibanaConsoleEditor
          ref="editorRef"
          v-model="doc"
          :meta="meta"
          @send="execute"
          @copy-curl="copyCurl"
          @active-change="onActiveChange"
        />
      </div>
      <div class="console-pane">
        <ConsoleResponseViewer
          :status="status"
          :duration="duration"
          :size="size"
          :body="response"
        />
      </div>
    </div>

    <UModal v-model:open="historyOpen" title="Request History" :ui="{ content: 'sm:max-w-4xl' }">
      <template #header>
        <div class="flex w-full items-center justify-between">
          <h3 class="text-base font-semibold">Request History</h3>
          <USelect v-model="methodFilter" :items="methodOptions" size="xs" class="w-28" />
        </div>
      </template>
      <template #body>
        <CerebroTable
          :data="filteredHistory"
          :columns="historyColumns"
          :default-sort="[{ id: 'created_at', desc: true }]"
          filter-placeholder="Filter history..."
          empty-text="No requests yet."
          @select="insertHistory"
        >
          <template #body-cell="{ row }">
            <span class="line-clamp-2 text-xs [overflow-wrap:anywhere]">{{ row.original.body }}</span>
          </template>
        </CerebroTable>
      </template>
    </UModal>

    <UModal v-model:open="shortcutsOpen" title="Keyboard Shortcuts">
      <template #header>
        <h3 class="text-base font-semibold">Keyboard Shortcuts</h3>
      </template>
      <template #body>
        <ul class="space-y-2 text-sm">
          <li><kbd>Ctrl/Cmd + Enter</kbd> — Send the active request</li>
          <li><kbd>Ctrl/Cmd + Shift + F</kbd> — Auto-indent the active request body</li>
          <li><kbd>Ctrl/Cmd + F</kbd> — Search in the editor / response</li>
          <li><kbd>Tab</kbd> — Accept the selected autocomplete suggestion</li>
          <li><kbd>▶</kbd> gutter — Send the request on that line</li>
        </ul>
      </template>
    </UModal>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import CerebroTable from '../../components/CerebroTable.vue'
import KibanaConsoleEditor from '../../components/rest/KibanaConsoleEditor.vue'
import ConsoleResponseViewer from '../../components/rest/ConsoleResponseViewer.vue'
import { useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { CerebroColumnDef } from '../../composables/useSortableColumns'
import type { RestMetadata } from '../../types/cerebro'
import { buildCurlCommand, parseConsoleDocument, type ParsedConsoleRequest } from '../../utils/consoleParser'
import { validateConsoleDocument } from '../../utils/consoleLinter'

const { clusterHost, clusterName } = useCluster()
const { apiCall } = useAPI()

const DEFAULT_BUFFER = `# Cerebro DevTools Console
GET _cluster/health

GET _cat/indices?v

POST _search
{
  "query": {
    "match_all": {}
  }
}
`

const doc = ref(DEFAULT_BUFFER)
const meta = ref<RestMetadata>({ indices: [], fieldsByIndex: {}, allFields: [], backendType: '', host: '' })
const response = ref<any>(null)
const status = ref(0)
const duration = ref(0)
const size = ref(0)
const sending = ref(false)
const sendAnyway = ref(false)
const lintErrors = ref<any[]>([])
const cursorRequest = ref<ParsedConsoleRequest | null>(null)
const history = ref<any[]>([])
const historyOpen = ref(false)
const shortcutsOpen = ref(false)
const methodFilter = ref('all')
const stacked = ref(false)
const editorRef = ref<any>(null)
const copyFeedback = ref('')

let copyTimer: ReturnType<typeof setTimeout> | null = null

const methodOptions = [
  { label: 'All', value: 'all' },
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' },
  { label: 'HEAD', value: 'HEAD' },
]

const activeRequest = computed<ParsedConsoleRequest | null>(() => {
  if (cursorRequest.value) return cursorRequest.value
  const requests = parseConsoleDocument(doc.value)
  return requests[0] || null
})

function onActiveChange(request: ParsedConsoleRequest) {
  cursorRequest.value = request
}

const filteredHistory = computed(() => methodFilter.value === 'all'
  ? history.value
  : history.value.filter((h) => h.method === methodFilter.value))

const historyColumns: CerebroColumnDef<any>[] = [
  { id: 'created_at', header: 'Time', accessorKey: 'created_at', class: 'whitespace-nowrap' },
  { id: 'method', header: 'Method', accessorKey: 'method', class: 'whitespace-nowrap' },
  { id: 'path', header: 'Path', accessorKey: 'path' },
  { id: 'body', header: 'Body', accessorKey: 'body' },
]

function loadBuffer() {
  if (typeof window === 'undefined') return
  try {
    const saved = window.localStorage.getItem('cerebro-console-buffer')
    if (saved) doc.value = saved
  } catch {
    // localStorage may be unavailable (private browsing/quota); ignore.
  }
}

function saveBuffer() {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem('cerebro-console-buffer', doc.value)
  } catch {
    // ignore
  }
}

async function loadSchema() {
  if (!clusterHost.value) return
  const resp = await apiCall('POST', '/rest', { host: clusterHost.value })
  if (resp?.status === 200) {
    meta.value = {
      indices: resp.body?.indices || [],
      fieldsByIndex: resp.body?.fieldsByIndex || {},
      allFields: resp.body?.allFields || [],
      backendType: resp.body?.backendType || '',
      host: resp.body?.host || clusterHost.value,
    }
  }
}

async function loadHistory() {
  if (!clusterHost.value) return
  const resp = await apiCall('POST', '/rest/history', { host: clusterHost.value })
  if (resp?.status === 200) history.value = resp.body || []
}

async function send() {
  if (!activeRequest.value) return
  await execute(activeRequest.value)
}

async function execute(request: ParsedConsoleRequest) {
  const errors = validateConsoleDocument(doc.value).filter(
    (d) => d.severity === 'error' && d.from >= request.from && d.to <= request.to,
  )
  lintErrors.value = errors
  if (errors.length && !sendAnyway.value) return
  sendAnyway.value = false

  let data = request.rawBody || undefined
  if (request.isNdjsonEndpoint && data && !data.endsWith('\n')) data += '\n'

  sending.value = true
  const started = performance.now()
  const resp = await apiCall('POST', '/rest/request', {
    host: clusterHost.value,
    method: request.method,
    path: request.path,
    data,
  }, { suppressErrorAlert: true })
  duration.value = Math.round(performance.now() - started)
  sending.value = false

  if (resp) {
    response.value = resp.body
    status.value = resp.status
    try {
      size.value = new Blob([JSON.stringify(resp.body)]).size
    } catch {
      size.value = 0
    }
    await loadHistory()
  }
}

function format() {
  editorRef.value?.formatActive()
}

function showCopyFeedback(message: string) {
  copyFeedback.value = message
  if (copyTimer) clearTimeout(copyTimer)
  copyTimer = setTimeout(() => { copyFeedback.value = '' }, 2000)
}

async function copyCurl(request?: ParsedConsoleRequest) {
  const target = request || activeRequest.value
  if (!target) return
  try {
    await navigator.clipboard.writeText(buildCurlCommand(meta.value.host || clusterHost.value, target))
    showCopyFeedback(`Copied cURL for ${target.method} ${target.path}`)
  } catch {
    showCopyFeedback('Could not copy to clipboard.')
  }
}

function insertHistory(row: any) {
  const data = row?.original ?? row
  const block = `${data.method} ${data.path}${data.body && data.body !== '{}' ? `\n${data.body}` : ''}`
  doc.value = `${doc.value.trimEnd()}\n\n${block}\n`
  historyOpen.value = false
}

onMounted(() => {
  loadBuffer()
  loadSchema()
  loadHistory()
  if (typeof window !== 'undefined') stacked.value = window.innerWidth < 900
})

// Persist the buffer on every edit (guarded for SSR/private browsing).
watch(doc, saveBuffer)

onBeforeUnmount(() => {
  if (copyTimer) clearTimeout(copyTimer)
})
</script>
