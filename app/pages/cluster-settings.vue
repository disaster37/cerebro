<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">Cluster Settings</h4>
      <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
        Refresh
      </UButton>
    </div>

    <div v-if="loading" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="cluster-settings-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else-if="settings">
      <UAlert v-if="saveSuccess" color="success" variant="soft" class="mb-3" data-testid="cluster-settings-success" :description="'Cluster settings saved.'" />
      <UAlert v-if="saveError" color="error" variant="soft" class="mb-3" data-testid="cluster-settings-save-error" :description="saveError" />

      <UCard class="mb-4">
        <template #header>
          <h5 class="text-sm font-semibold">Add / Set Extra Arbitrary Setting</h5>
        </template>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-4">
          <UFormField label="Setting Key">
            <UInput v-model="extraKey" list="known-settings" placeholder="cluster.max_shards_per_node" class="w-full" data-testid="extra-setting-key" />
            <datalist id="known-settings">
              <option v-for="key in knownKeyList" :key="key" :value="key" />
            </datalist>
          </UFormField>
          <UFormField label="Value">
            <UInput v-model="extraValue" placeholder="Reset to default when empty" class="w-full" data-testid="extra-setting-value" />
          </UFormField>
          <UFormField label="Scope">
            <USelect v-model="extraScope" :items="scopeOptions" class="w-full" />
          </UFormField>
          <UFormField label="&nbsp;">
            <UButton block :disabled="!extraValidation.valid" data-testid="stage-extra-setting" @click="stageExtra">
              Stage Setting
            </UButton>
          </UFormField>
        </div>
        <p v-if="extraValidation.message" class="mt-2 text-xs" :class="extraValidation.valid ? 'text-amber-500' : 'text-error-500'" data-testid="extra-setting-validation">
          {{ extraValidation.message }}
        </p>
      </UCard>

      <div class="mb-3 flex flex-wrap items-center gap-3">
        <UInput v-model="search" size="sm" icon="i-lucide-search" placeholder="Filter settings by name..." class="w-64" />
        <UCheckbox v-model="showAllDefaults" label="Show all cluster defaults (including static read-only)" />
        <USelect v-model="defaultScope" :items="scopeOptions" size="sm" class="w-64" />
      </div>

      <div class="space-y-4">
        <UCard v-for="group in visibleGroups" :key="group.title">
          <template #header>
            <h5 class="text-sm font-semibold">{{ group.title }}</h5>
          </template>
          <div class="space-y-3">
            <div v-for="setting in group.settings" :key="setting.key" class="grid grid-cols-1 items-center gap-2 md:grid-cols-[1fr_18rem]">
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium">{{ setting.label }}</span>
                  <UBadge v-if="sourceOf(setting.key)" color="neutral" variant="soft">{{ sourceOf(setting.key) }}</UBadge>
                  <UBadge v-if="isOverridden(setting.key)" color="info" variant="soft">override</UBadge>
                </div>
                <code class="text-[11px] text-neutral-500">{{ setting.key }}</code>
              </div>
              <div>
                <USwitch
                  v-if="setting.type === 'boolean'"
                  :model-value="Boolean(displayValue(setting.key))"
                  @update:model-value="stage(setting.key, $event, defaultScope)"
                />
                <USelect
                  v-else-if="setting.type === 'enum'"
                  :model-value="String(displayValue(setting.key) ?? '')"
                  :items="setting.options || []"
                  class="w-full"
                  @update:model-value="stage(setting.key, $event, defaultScope)"
                />
                <UInput
                  v-else
                  :model-value="String(displayValue(setting.key) ?? '')"
                  class="w-full"
                  @update:model-value="stage(setting.key, $event, defaultScope)"
                />
              </div>
            </div>
          </div>
        </UCard>
      </div>

      <UCard v-if="staged.length" class="sticky bottom-2 mt-4 ring-1 ring-primary-500/40" data-testid="pending-changes">
        <template #header>
          <div class="flex items-center justify-between">
            <h5 class="text-sm font-semibold">Pending Changes ({{ staged.length }})</h5>
            <UButton size="sm" :loading="saving" data-testid="save-cluster-settings" @click="save">Save Changes</UButton>
          </div>
        </template>
        <div class="space-y-3">
          <div v-for="change in staged" :key="change.key" class="rounded-lg border border-neutral-200 p-2 dark:border-neutral-800">
            <div class="flex flex-wrap items-center gap-2">
              <code class="text-xs">{{ change.key }}</code>
              <span class="text-xs text-neutral-500">{{ formatValue(effectiveValue(change.key)) }} →</span>
              <UBadge :color="change.reset ? 'warning' : 'success'" variant="soft">
                {{ change.reset ? 'RESET (null)' : formatValue(change.value) }}
              </UBadge>
              <USelect v-model="change.scope" :items="scopeOptions" size="xs" class="w-56" />
              <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-undo-2" @click="revert(change.key)">Revert</UButton>
            </div>
            <div v-if="change.scope === 'persistent' && hasTransientShadow(change.key)" class="mt-2">
              <UCheckbox v-model="change.clearTransient" label="Also clear active transient override (transient[key] = null) so persistent value takes effect immediately" />
            </div>
          </div>
        </div>
      </UCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { extractEnvelopeError, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'
import type { ClusterSettingsResponse } from '../types/cerebro'
import {
  CLUSTER_SETTINGS_CATALOG,
  validateSettingKey,
  type ClusterSettingDef,
} from '../utils/clusterSettingsCatalog'

interface StagedChange {
  key: string
  value: any
  scope: 'persistent' | 'transient'
  reset: boolean
  clearTransient: boolean
}

const { clusterHost } = useCluster()
const { apiCall } = useAPI()

const settings = ref<ClusterSettingsResponse | null>(null)
const loading = ref(true)
const error = ref('')
const search = ref('')
const showAllDefaults = ref(false)
const defaultScope = ref<'persistent' | 'transient'>('persistent')
const staged = ref<StagedChange[]>([])
const extraKey = ref('')
const extraValue = ref('')
const extraScope = ref<'persistent' | 'transient'>('persistent')
const saving = ref(false)
const saveError = ref('')
const saveSuccess = ref(false)

const scopeOptions = [
  { label: 'Persistent (survives cluster reboot)', value: 'persistent' },
  { label: 'Transient (resets on full cluster restart)', value: 'transient' },
]

function flattenKeys(obj: any, prefix = '', out = new Set<string>()): Set<string> {
  for (const [key, value] of Object.entries(obj || {})) {
    const full = prefix ? `${prefix}.${key}` : key
    out.add(full)
    if (value && typeof value === 'object' && !Array.isArray(value)) flattenKeys(value, full, out)
  }
  return out
}

const knownKeys = computed(() => {
  const out = new Set<string>()
  flattenKeys(settings.value?.defaults, '', out)
  flattenKeys(settings.value?.persistent, '', out)
  flattenKeys(settings.value?.transient, '', out)
  return out
})

const knownKeyList = computed(() => [...knownKeys.value].sort())

function lookup(obj: any, key: string): any {
  if (!obj) return undefined
  if (Object.prototype.hasOwnProperty.call(obj, key)) return obj[key]
  let current = obj
  for (const part of key.split('.')) {
    if (current === null || typeof current !== 'object' || !(part in current)) return undefined
    current = current[part]
  }
  return current
}

function effectiveValue(key: string): any {
  const transient = lookup(settings.value?.transient, key)
  if (transient !== undefined) return transient
  const persistent = lookup(settings.value?.persistent, key)
  if (persistent !== undefined) return persistent
  return lookup(settings.value?.defaults, key)
}

function sourceOf(key: string): string {
  if (lookup(settings.value?.transient, key) !== undefined) return 'transient'
  if (lookup(settings.value?.persistent, key) !== undefined) return 'persistent'
  if (lookup(settings.value?.defaults, key) !== undefined) return 'default'
  return ''
}

function isOverridden(key: string): boolean {
  return lookup(settings.value?.persistent, key) !== undefined || lookup(settings.value?.transient, key) !== undefined
}

function hasTransientShadow(key: string): boolean {
  return lookup(settings.value?.transient, key) !== undefined
}

function displayValue(key: string): any {
  const change = staged.value.find((c) => c.key === key)
  if (change) return change.reset ? '' : change.value
  return effectiveValue(key)
}

function formatValue(value: any): string {
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function parseValue(setting: ClusterSettingDef | undefined, raw: any): any {
  if (raw === '' || raw === null || raw === undefined) return null
  if (setting?.type === 'boolean') return raw === true || raw === 'true'
  if (setting?.type === 'number') {
    const num = Number(raw)
    return Number.isNaN(num) ? raw : num
  }
  return raw
}

function findSetting(key: string): ClusterSettingDef | undefined {
  for (const group of CLUSTER_SETTINGS_CATALOG) {
    const found = group.settings.find((s) => s.key === key)
    if (found) return found
  }
  return undefined
}

function stage(key: string, raw: any, scope: 'persistent' | 'transient') {
  const value = parseValue(findSetting(key), raw)
  const reset = value === null
  const existing = staged.value.find((c) => c.key === key)
  if (existing) {
    existing.value = value
    existing.reset = reset
    existing.scope = scope
  } else {
    staged.value.push({ key, value, scope, reset, clearTransient: true })
  }
  saveSuccess.value = false
}

function revert(key: string) {
  staged.value = staged.value.filter((c) => c.key !== key)
}

const extraValidation = computed(() => validateSettingKey(extraKey.value, knownKeys.value))

function stageExtra() {
  if (!extraValidation.value.valid) return
  stage(extraKey.value.trim(), extraValue.value, extraScope.value)
  extraKey.value = ''
  extraValue.value = ''
}

const visibleGroups = computed(() => {
  const groups = CLUSTER_SETTINGS_CATALOG.map((group) => ({
    title: group.title,
    settings: group.settings.filter((setting) => {
      if (search.value && !setting.key.toLowerCase().includes(search.value.toLowerCase())) return false
      if (showAllDefaults.value) return true
      return knownKeys.value.has(setting.key) || isOverridden(setting.key)
    }),
  })).filter((group) => group.settings.length > 0)

  const catalogKeys = new Set(CLUSTER_SETTINGS_CATALOG.flatMap((g) => g.settings.map((s) => s.key)))
  const otherKeys = [...knownKeys.value].filter((key) => !catalogKeys.has(key) && isOverridden(key))
  if (otherKeys.length) {
    groups.push({
      title: 'Other Active Overrides',
      settings: otherKeys.map((key) => ({ key, label: key, type: 'string' as const })),
    })
  }
  return groups
})

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/cluster_settings', { host: clusterHost.value })
  if (resp?.status === 200) {
    settings.value = resp.body || {}
    error.value = ''
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load cluster settings (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load cluster settings (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

async function save() {
  saving.value = true
  saveError.value = ''
  saveSuccess.value = false

  const persistent: Record<string, any> = {}
  const transient: Record<string, any> = {}
  for (const change of staged.value) {
    const target = change.scope === 'persistent' ? persistent : transient
    target[change.key] = change.reset ? null : change.value
    if (change.scope === 'persistent' && change.clearTransient && hasTransientShadow(change.key)) {
      transient[change.key] = null
    }
  }

  const payload: Record<string, any> = {}
  if (Object.keys(persistent).length) payload.persistent = persistent
  if (Object.keys(transient).length) payload.transient = transient
  if (!Object.keys(payload).length) {
    payload.persistent = persistent
    payload.transient = transient
  }

  const resp = await apiCall('POST', '/cluster_settings/save', {
    host: clusterHost.value,
    settings: payload,
  }, { suppressErrorAlert: true })

  if (resp?.status === 200) {
    staged.value = []
    saveSuccess.value = true
    await load()
  } else {
    saveError.value = extractEnvelopeError(resp?.body) || `Failed to save cluster settings (${resp?.status || 'network error'}).`
  }
  saving.value = false
}

onMounted(load)
</script>
