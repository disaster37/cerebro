<template>
  <UModal v-model:open="isOpen" :title="`Restore snapshot ${snapshot?.snapshot || ''}`">
    <template #header>
      <h3 class="text-base font-semibold">Restore snapshot {{ snapshot?.snapshot }}</h3>
    </template>
    <template #body>
      <div class="max-h-[70vh] space-y-5 overflow-y-auto pr-1">
        <section>
          <h4 class="mb-2 text-sm font-semibold">1. Index selection</h4>
          <div class="space-y-2">
            <UCheckbox v-model="restoreAll" label="Restore all regular indices" />
            <UCheckbox v-model="selectSpecific" label="Select specific indices" />
          </div>
          <div v-if="selectSpecific" class="mt-3 space-y-2">
            <div class="flex flex-wrap items-center gap-2">
              <UInput v-model="search" size="sm" icon="i-lucide-search" placeholder="Filter indices..." class="w-56" />
              <UCheckbox v-model="showSystem" label="Show system/hidden indices (.*)" />
              <UButton size="xs" color="neutral" variant="outline" @click="selectAll">Select All</UButton>
              <UButton size="xs" color="neutral" variant="ghost" @click="selected = []">Clear</UButton>
            </div>
            <div class="max-h-48 overflow-y-auto rounded-lg border border-neutral-200 p-2 dark:border-neutral-800">
              <UCheckbox
                v-for="index in visibleIndices"
                :key="index"
                :model-value="selected.includes(index)"
                :label="index"
                @update:model-value="toggleIndex(index, $event)"
              />
            </div>
          </div>
        </section>

        <section>
          <h4 class="mb-2 text-sm font-semibold">2. Rename indices & conflict check</h4>
          <UCheckbox v-model="renameEnabled" label="Rename indices on restore" />
          <div v-if="renameEnabled" class="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
            <UFormField label="Rename pattern">
              <UInput v-model="renamePattern" placeholder="(.+)" class="w-full" data-testid="restore-rename-pattern" />
            </UFormField>
            <UFormField label="Rename replacement">
              <UInput v-model="renameReplacement" placeholder="restored_$1" class="w-full" data-testid="restore-rename-replacement" />
            </UFormField>
          </div>
          <UAlert v-if="conflicts.length" color="warning" variant="soft" class="mt-2" data-testid="restore-conflicts">
            <template #description>
              Target index name(s) already exist or collide: {{ conflicts.join(', ') }}. Close/delete the open index or rename on restore.
            </template>
          </UAlert>
        </section>

        <section>
          <h4 class="mb-2 text-sm font-semibold">3. Restore options</h4>
          <div class="space-y-2">
            <UCheckbox v-model="includeAliases" label="Include aliases" />
            <UCheckbox v-model="ignoreUnavailable" label="Ignore unavailable indices" />
            <UCheckbox v-model="partial" label="Restore partial snapshot" />
            <p v-if="snapshot?.state === 'PARTIAL'" class="text-xs text-amber-500">
              This snapshot is PARTIAL; enable partial restore to recover the available shards.
            </p>
            <UCheckbox
              v-model="includeGlobalState"
              label="Include global cluster state"
              :disabled="snapshot?.include_global_state === false"
            />
            <p v-if="snapshot?.include_global_state === false" class="text-xs text-neutral-500">
              This snapshot was taken without global cluster state.
            </p>
          </div>

          <details class="mt-3">
            <summary class="cursor-pointer text-sm font-medium">Index Settings Modifiers</summary>
            <div class="mt-2 space-y-2">
              <UFormField label="Override index settings (JSON)">
                <UTextarea v-model="indexSettingsText" :rows="3" class="w-full font-mono" placeholder='{"index.number_of_replicas": 0}' />
              </UFormField>
              <UFormField label="Ignore index settings (comma-separated)">
                <UInput v-model="ignoreIndexSettingsText" class="w-full" placeholder="index.refresh_interval" />
              </UFormField>
            </div>
          </details>
        </section>

        <UAlert v-if="validationError" color="error" variant="soft" :description="validationError" data-testid="restore-validation" />
        <UAlert v-if="error" color="error" variant="soft" data-testid="restore-error">
          <template #description>
            <div class="space-y-2">
              <p>{{ error }}</p>
              <details v-if="rawError">
                <summary class="cursor-pointer text-xs">Raw error JSON</summary>
                <pre class="mt-1 max-h-40 overflow-auto rounded bg-neutral-100 p-2 text-[11px] dark:bg-neutral-900">{{ rawError }}</pre>
              </details>
            </div>
          </template>
        </UAlert>
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="isOpen = false">Cancel</UButton>
        <UButton :loading="submitting" :disabled="Boolean(validationError)" data-testid="restore-submit" @click="submit">
          Restore
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { extractEnvelopeError, useAPI } from '../../composables/useAPI'
import { useCluster } from '../../composables/useCluster'
import type { SnapshotEntry } from '../../types/cerebro'

const props = defineProps<{
  open: boolean
  snapshot: SnapshotEntry | null
  clusterIndices: string[]
  repository: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'restored'): void
}>()

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const restoreAll = ref(true)
const selectSpecific = ref(false)
const selected = ref<string[]>([])
const showSystem = ref(false)
const search = ref('')
const renameEnabled = ref(false)
const renamePattern = ref('')
const renameReplacement = ref('')
const includeAliases = ref(true)
const ignoreUnavailable = ref(true)
const partial = ref(false)
const includeGlobalState = ref(false)
const indexSettingsText = ref('')
const ignoreIndexSettingsText = ref('')
const error = ref('')
const rawError = ref('')
const submitting = ref(false)

const isOpen = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
})

watch(() => props.open, (open) => {
  if (!open) return
  restoreAll.value = true
  selectSpecific.value = false
  selected.value = []
  showSystem.value = false
  search.value = ''
  renameEnabled.value = false
  renamePattern.value = ''
  renameReplacement.value = ''
  includeAliases.value = true
  ignoreUnavailable.value = true
  partial.value = false
  includeGlobalState.value = false
  indexSettingsText.value = ''
  ignoreIndexSettingsText.value = ''
  error.value = ''
  rawError.value = ''
})

watch(restoreAll, (value) => {
  if (value) selectSpecific.value = false
})
watch(selectSpecific, (value) => {
  if (value) restoreAll.value = false
})

const regularIndices = computed(() => (props.snapshot?.indices || []).filter((i) => !i.startsWith('.')))

const visibleIndices = computed(() => (props.snapshot?.indices || []).filter((index) => {
  if (!showSystem.value && index.startsWith('.')) return false
  if (search.value && !index.toLowerCase().includes(search.value.toLowerCase())) return false
  return true
}))

const indicesToRestore = computed(() => {
  if (selectSpecific.value) return selected.value
  if (restoreAll.value) return showSystem.value ? (props.snapshot?.indices || []) : regularIndices.value
  return []
})

function renamed(name: string): string {
  if (!renameEnabled.value || !renamePattern.value || !renameReplacement.value) return name
  try {
    return name.replace(new RegExp(renamePattern.value), renameReplacement.value)
  } catch {
    return name
  }
}

const conflicts = computed(() => {
  const targets = indicesToRestore.value.map(renamed)
  const open = new Set(props.clusterIndices || [])
  const seen = new Set<string>()
  const out = new Set<string>()
  for (const target of targets) {
    if (open.has(target)) out.add(target)
    if (seen.has(target)) out.add(target)
    seen.add(target)
  }
  return [...out]
})

const validationError = computed(() => {
  if (selectSpecific.value && selected.value.length === 0) return 'Select at least one index to restore.'
  if (renameEnabled.value) {
    if (!renamePattern.value || !renameReplacement.value) return 'Both rename pattern and replacement are required.'
    try {
      new RegExp(renamePattern.value)
    } catch {
      return 'Rename pattern is not a valid regular expression.'
    }
  }
  if (indexSettingsText.value.trim()) {
    try {
      const parsed = JSON.parse(indexSettingsText.value)
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        return 'Index settings override must be a JSON object.'
      }
    } catch {
      return 'Index settings override must be valid JSON.'
    }
  }
  return ''
})

function toggleIndex(index: string, checked: boolean) {
  if (checked) {
    if (!selected.value.includes(index)) selected.value.push(index)
  } else {
    selected.value = selected.value.filter((i) => i !== index)
  }
}

function selectAll() {
  selected.value = visibleIndices.value.slice()
}

async function submit() {
  if (validationError.value || !props.snapshot) return
  error.value = ''
  rawError.value = ''
  submitting.value = true

  const body: Record<string, any> = {
    host: clusterHost.value,
    repository: props.repository,
    snapshot: props.snapshot.snapshot,
    includeAliases: includeAliases.value,
    ignoreUnavailable: ignoreUnavailable.value,
    partial: partial.value,
    includeGlobalState: includeGlobalState.value,
  }
  if (selectSpecific.value) body.indices = selected.value
  else if (restoreAll.value && !showSystem.value) body.indices = regularIndices.value
  if (renameEnabled.value) {
    body.renamePattern = renamePattern.value
    body.renameReplacement = renameReplacement.value
  }
  if (indexSettingsText.value.trim()) body.indexSettings = JSON.parse(indexSettingsText.value)
  if (ignoreIndexSettingsText.value.trim()) {
    body.ignoreIndexSettings = ignoreIndexSettingsText.value.split(',').map((s) => s.trim()).filter(Boolean)
  }

  const resp = await apiCall('POST', '/snapshots/restore', body, { suppressErrorAlert: true })
  submitting.value = false
  if (resp?.status === 200) {
    isOpen.value = false
    emit('restored')
  } else {
    error.value = extractEnvelopeError(resp?.body) || `Restore failed (${resp?.status || 'network error'}).`
    try {
      rawError.value = JSON.stringify(resp?.body, null, 2)
    } catch {
      rawError.value = String(resp?.body)
    }
  }
}
</script>
