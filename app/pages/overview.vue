<template>
  <div>
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <h4 class="text-lg font-semibold">
        Overview
        <small class="text-neutral-500">{{ data?.cluster_name || clusterName }}</small>
        <UBadge v-if="data?.status" :color="healthColor" variant="soft" class="ml-2">{{ data.status }}</UBadge>
      </h4>
      <div class="flex items-center gap-2">
        <USelect v-model="autoRefresh" :items="refreshOptions" size="sm" class="w-24" />
        <UButton size="sm" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="loading" @click="load">
          Refresh
        </UButton>
        <UButton size="sm" color="neutral" variant="ghost" :icon="chartsOpen ? 'i-lucide-chevron-up' : 'i-lucide-chart-pie'" @click="chartsOpen = !chartsOpen">
          Charts
        </UButton>
      </div>
    </div>

    <div v-if="loading && !data" class="flex justify-center p-5" role="status" aria-label="Loading">
      <UIcon name="i-lucide-loader-circle" class="size-6 animate-spin" />
    </div>
    <UCard v-else-if="error" class="mt-3 ring-1 ring-error-200" data-testid="overview-error">
      <p class="mb-3 font-medium text-error-600">{{ error }}</p>
      <UButton to="/connect" size="sm" color="primary" variant="outline">Connect to Cluster</UButton>
    </UCard>

    <div v-else-if="data">
      <div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <UCard v-for="stat in summary" :key="stat.label" class="text-center">
          <p class="text-xl font-semibold">{{ stat.value }}</p>
          <p class="text-xs text-neutral-500">{{ stat.label }}</p>
        </UCard>
      </div>

      <div v-if="chartsOpen" class="mb-6">
        <ClusterMetricsCharts :data="data" />
      </div>

      <div class="mb-3 flex flex-wrap items-center gap-3">
        <UInput v-model="indexFilter" size="sm" icon="i-lucide-search" placeholder="Filter indices / aliases..." class="w-64" />
        <UCheckbox v-model="showClosed" :label="`closed (${data.closed_indices})`" />
        <UCheckbox v-model="showSpecial" :label="specialLabel" />
        <UCheckbox v-model="showBacking" label="backing (.ds)" />
        <UCheckbox v-model="unhealthyOnly" label="unhealthy only" :disabled="!hasUnhealthyShards" />
        <UInput v-model="nodeFilter" size="sm" icon="i-lucide-server" placeholder="Filter nodes..." class="w-48" />
        <div class="flex items-center gap-1">
          <UButton
            v-for="role in roleOptions"
            :key="role.value"
            size="xs"
            :color="roleFilter[role.value] ? 'primary' : 'neutral'"
            :variant="roleFilter[role.value] ? 'solid' : 'outline'"
            @click="roleFilter[role.value] = !roleFilter[role.value]"
          >
            {{ role.label }}
          </UButton>
        </div>
        <UBadge
          v-if="dataStreamFilter"
          color="primary"
          variant="soft"
          class="cursor-pointer"
          data-testid="overview-datastream-filter"
          @click="dataStreamFilter = ''"
        >
          Data stream: {{ dataStreamFilter }} [Clear]
        </UBadge>
      </div>

      <UAlert
        v-if="unhealthyIndices.length"
        color="warning"
        variant="soft"
        class="mb-3"
        data-testid="unhealthy-banner"
      >
        <template #description>
          <div class="flex flex-wrap items-center gap-2">
            <span>{{ unhealthyIndices.length }} indices have unassigned or initializing shards.</span>
            <UButton size="xs" color="warning" variant="soft" @click="focusUnhealthy">
              Focus unhealthy indices
            </UButton>
          </div>
        </template>
      </UAlert>

      <div class="overflow-x-auto rounded-lg ring-1 ring-neutral-200 dark:ring-neutral-800">
        <table class="w-full table-fixed border-collapse text-xs">
          <thead>
            <tr class="border-b border-neutral-200 dark:border-neutral-800">
              <th class="w-64 p-2 text-left align-top">
                <div class="flex flex-wrap items-center gap-1">
                  <UDropdownMenu :items="allocationMenu">
                    <UButton
                      size="xs"
                      :color="data.shard_allocation ? 'success' : 'error'"
                      variant="soft"
                      :icon="data.shard_allocation ? 'i-lucide-lock-open' : 'i-lucide-lock'"
                    >
                      {{ data.shard_allocation ? 'Allocation on' : 'Allocation off' }}
                    </UButton>
                  </UDropdownMenu>
                  <UButton size="xs" color="neutral" variant="ghost" :icon="expandedView ? 'i-lucide-minimize-2' : 'i-lucide-maximize-2'" @click="expandedView = !expandedView" />
                  <UButton size="xs" color="neutral" variant="ghost" :icon="indexSortDir === 'asc' ? 'i-lucide-arrow-up' : 'i-lucide-arrow-down'" @click="indexSortDir = indexSortDir === 'asc' ? 'desc' : 'asc'" />
                  <USelect v-model="indexSortKey" :items="sortKeys" size="xs" class="w-28" />
                  <UDropdownMenu :items="bulkMenu">
                    <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-list-checks">Bulk</UButton>
                  </UDropdownMenu>
                </div>
                <div class="mt-2 flex items-center gap-1">
                  <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevron-left" :disabled="page === 0" @click="page--" />
                  <span class="text-[11px] text-neutral-500">{{ page + 1 }} / {{ pageCount }}</span>
                  <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-chevron-right" :disabled="page >= pageCount - 1" @click="page++" />
                  <USelect v-model="pageSize" :items="pageSizes" size="xs" class="w-16" />
                </div>
              </th>
              <th
                v-for="idx in pagedIndices"
                :key="idx.name"
                class="min-w-40 border-l border-neutral-200 p-2 text-left align-top dark:border-neutral-800"
                :class="indexHeaderClass(idx)"
              >
                <div class="flex items-center justify-between gap-1">
                  <span class="truncate font-semibold" :title="idx.name">{{ idx.name }}</span>
                  <UDropdownMenu :items="indexMenu(idx)">
                    <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-ellipsis-vertical" aria-label="Index actions" />
                  </UDropdownMenu>
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-1">
                  <UBadge
                    v-if="idx.status === 'red' || idx.status === 'yellow'"
                    :color="idx.status === 'red' ? 'error' : 'warning'"
                    variant="soft"
                  >
                    {{ idx.status }} ({{ idx.unassigned_shard_count }} unassigned)
                  </UBadge>
                  <UBadge
                    v-if="idx.unreplicated"
                    color="warning"
                    variant="outline"
                    title="number_of_replicas is 0 (no replica copies)"
                  >
                    0 replicas
                  </UBadge>
                  <UBadge
                    v-if="idx.data_stream"
                    color="primary"
                    variant="soft"
                    class="cursor-pointer"
                    @click="filterByDataStream(idx.data_stream)"
                  >
                    DS: {{ idx.data_stream }}
                  </UBadge>
                  <UBadge v-if="idx.is_write_index" color="info" variant="soft">write</UBadge>
                  <UBadge v-if="idx.aliases?.length" color="info" variant="soft" :title="idx.aliases.join(', ')">
                    {{ idx.aliases[0] }}<span v-if="idx.aliases.length > 1"> (+{{ idx.aliases.length - 1 }})</span>
                  </UBadge>
                  <UBadge v-if="idx.closed" color="neutral" variant="soft">index closed</UBadge>
                </div>
                <div class="mt-1 text-[11px] text-neutral-500">
                  shards: {{ idx.num_shards }} * {{ idx.num_replicas + 1 }} | docs: {{ formatNumber(idx.doc_count) }} | size: {{ formatBytes(idx.size_in_bytes) }}
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="hasUnhealthyShards" class="border-b border-neutral-200 bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-900/60">
              <td class="p-2 align-top">
                <div class="font-medium">Unassigned / relocating</div>
                <div class="text-[11px] text-neutral-500">
                  unassigned: {{ data.unassigned_shards }} | initializing: {{ data.initializing_shards }} | relocating: {{ data.relocating_shards }}
                </div>
              </td>
              <td v-for="idx in pagedIndices" :key="`unassigned-${idx.name}`" class="border-l border-neutral-200 p-2 align-top dark:border-neutral-800">
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="(shard, sIdx) in (idx.shards?.unassigned || [])"
                    :key="`${shard.shard}-${shard.primary ? 'p' : 'r'}-${shard.state}-${sIdx}`"
                    class="shard-badge"
                    :class="shard.primary ? 'shard-badge--unassigned-primary shard-badge--primary' : 'shard-badge--unassigned shard-badge--replica'"
                    :title="`${shard.primary ? 'PRIMARY' : 'REPLICA'} shard ${shard.shard} ${shard.state}${shard.unassigned_info?.reason ? ` — ${shard.unassigned_info.reason}` : ''}`"
                    @click="inspectShard(idx, shard)"
                  >
                    {{ shard.shard }}
                  </button>
                </div>
              </td>
            </tr>

            <tr v-for="node in filteredNodes" :key="node.id" class="border-b border-neutral-200 dark:border-neutral-800">
              <td class="p-2 align-top">
                <div class="flex items-center gap-1">
                  <UIcon
                    :name="node.current_master ? 'i-lucide-star' : 'i-lucide-star'"
                    :class="node.current_master ? 'text-amber-500' : 'text-neutral-400'"
                    :style="node.current_master ? '' : 'opacity:.35'"
                  />
                  <UBadge v-if="node.master" color="primary" variant="soft">M</UBadge>
                  <UBadge v-if="node.data" color="success" variant="soft">D</UBadge>
                  <UBadge v-if="node.ingest" color="info" variant="soft">I</UBadge>
                  <UBadge v-if="node.coordinating" color="neutral" variant="soft">C</UBadge>
                </div>
                <button class="mt-1 block truncate text-left font-medium hover:underline" @click="inspectNode(node)">
                  {{ node.name }}
                </button>
                <div class="text-[11px] text-neutral-500">{{ node.host }} / {{ node.ip }}</div>
                <div v-if="Object.keys(node.attributes || {}).length" class="mt-1 flex flex-wrap gap-1">
                  <UBadge v-for="(value, key) in node.attributes" :key="key" color="neutral" variant="soft">
                    {{ key }}={{ value }}
                  </UBadge>
                </div>
                <div v-if="expandedView" class="mt-1 flex flex-wrap gap-1">
                  <UBadge color="neutral" variant="soft">JVM {{ node.jvm_version }}</UBadge>
                  <UBadge color="neutral" variant="soft">ES/OS {{ node.es_version }}</UBadge>
                </div>
                <div class="mt-2">
                  <NodeResourceBars :node="node" />
                </div>
              </td>
              <td v-for="idx in pagedIndices" :key="`${node.id}-${idx.name}`" class="border-l border-neutral-200 p-2 align-top dark:border-neutral-800">
                <div class="flex flex-wrap gap-1">
                  <UDropdownMenu
                    v-for="shard in (idx.shards?.[node.id] || [])"
                    :key="`${shard.shard}-${shard.primary}-${shard.state}`"
                    :items="shardMenu(idx, shard)"
                  >
                    <button
                      class="shard-badge"
                      :class="[shardClass(shard), shard.primary ? 'shard-badge--primary' : 'shard-badge--replica']"
                      :title="`Shard ${shard.shard} ${shard.state}${shard.primary ? ' (primary)' : ''}`"
                    >
                      {{ shard.shard }}
                    </button>
                  </UDropdownMenu>
                  <UButton
                    v-if="canRelocateTo(idx, node)"
                    size="xs"
                    color="primary"
                    variant="outline"
                    icon="i-lucide-download"
                    title="Relocate shard here"
                    @click="relocate(idx, node)"
                  />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <JsonInspectorModal v-model:open="inspectorOpen" :title="inspectorTitle" :data="inspectorData" />
    <ConfirmActionModal
      v-model:open="confirmOpen"
      :title="confirmTitle"
      :description="confirmDescription"
      :confirm-label="confirmLabel"
      @confirm="runConfirmed"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import ClusterMetricsCharts from '../components/charts/ClusterMetricsCharts.vue'
import NodeResourceBars from '../components/overview/NodeResourceBars.vue'
import JsonInspectorModal from '../components/JsonInspectorModal.vue'
import ConfirmActionModal from '../components/ConfirmActionModal.vue'
import { extractEnvelopeError, pushAlert, useAPI } from '../composables/useAPI'
import { useCluster } from '../composables/useCluster'
import type { OverviewData, OverviewIndex, OverviewNode, OverviewShard } from '../types/cerebro'
import { formatBytes, formatNumber } from '../utils/format'

const { clusterHost, clusterName } = useCluster()
const { apiCall } = useAPI()

const data = ref<OverviewData | null>(null)
const loading = ref(true)
const error = ref('')
const chartsOpen = ref(false)
const expandedView = ref(false)
const indexFilter = ref('')
const nodeFilter = ref('')
const showClosed = ref(false)
const showSpecial = ref(false)
const showBacking = ref(true)
const unhealthyOnly = ref(false)
const dataStreamFilter = ref('')
const roleFilter = ref<Record<string, boolean>>({ M: true, D: true, I: true, C: true })
const indexSortKey = ref('name')
const indexSortDir = ref<'asc' | 'desc'>('asc')
const page = ref(0)
const pageSize = ref(8)
const autoRefresh = ref('off')
const relocatingShard = ref<{ index: string; shard: OverviewShard } | null>(null)
const inspectorOpen = ref(false)
const inspectorTitle = ref('')
const inspectorData = ref<any>(null)
const confirmOpen = ref(false)
const confirmTitle = ref('')
const confirmDescription = ref('')
const confirmLabel = ref('Confirm')
const pendingAction = ref<null | (() => Promise<void>)>(null)

let timer: ReturnType<typeof setInterval> | null = null

const refreshOptions = [
  { label: 'Off', value: 'off' },
  { label: '5s', value: '5' },
  { label: '15s', value: '15' },
  { label: '30s', value: '30' },
  { label: '60s', value: '60' },
]
const pageSizes = [5, 8, 12]
const sortKeys = [
  { label: 'Name', value: 'name' },
  { label: 'Docs', value: 'doc_count' },
  { label: 'Size', value: 'size_in_bytes' },
  { label: 'Unhealthy', value: 'unhealthy' },
]
const roleOptions = [
  { label: 'M', value: 'M' },
  { label: 'D', value: 'D' },
  { label: 'I', value: 'I' },
  { label: 'C', value: 'C' },
]

const healthColor = computed(() => {
  switch (data.value?.status) {
    case 'green': return 'success'
    case 'yellow': return 'warning'
    case 'red': return 'error'
    default: return 'neutral'
  }
})

const summary = computed(() => {
  if (!data.value) return []
  const totalShards = (data.value.active_shards || 0) + (data.value.initializing_shards || 0) + (data.value.relocating_shards || 0) + (data.value.unassigned_shards || 0)
  return [
    { label: 'Nodes', value: formatNumber(data.value.number_of_nodes) },
    { label: 'Indices', value: formatNumber(data.value.total_indices) },
    { label: 'Total Shards', value: formatNumber(totalShards) },
    { label: 'Docs', value: formatNumber(data.value.docs_count) },
    { label: 'Store Size', value: formatBytes(data.value.size_in_bytes) },
    { label: 'Unassigned', value: formatNumber(data.value.unassigned_shards) },
  ]
})

const hasUnhealthyShards = computed(() => Boolean(
  data.value && ((data.value.unassigned_shards || 0) > 0 || (data.value.relocating_shards || 0) > 0 || (data.value.initializing_shards || 0) > 0),
))

const unhealthyIndices = computed(() => (data.value?.indices || []).filter((i) => !i.closed && i.unhealthy))

const specialLabel = computed(() => {
  const total = data.value?.special_indices || 0
  const unhealthy = (data.value?.indices || []).filter((i) => i.special && i.unhealthy).length
  return unhealthy ? `.special (${total}, ${unhealthy} unhealthy)` : `.special (${total})`
})

const filteredIndices = computed(() => {
  let indices = data.value?.indices || []
  if (!showClosed.value) indices = indices.filter((i) => !i.closed)
  if (!showSpecial.value) indices = indices.filter((i) => !i.special || (showBacking.value && i.backing_index))
  if (!showBacking.value) indices = indices.filter((i) => !i.backing_index)
  if (unhealthyOnly.value) indices = indices.filter((i) => i.unhealthy)
  if (dataStreamFilter.value) indices = indices.filter((i) => i.data_stream === dataStreamFilter.value)
  if (indexFilter.value) {
    let matcher: (value: string) => boolean
    try {
      const re = new RegExp(indexFilter.value, 'i')
      matcher = (value: string) => re.test(value)
    } catch {
      const needle = indexFilter.value.toLowerCase()
      matcher = (value: string) => value.toLowerCase().includes(needle)
    }
    indices = indices.filter((i) => matcher(i.name) || (i.aliases || []).some((a) => matcher(a)))
  }
  return indices
})

const sortedIndices = computed(() => {
  const key = indexSortKey.value
  const dir = indexSortDir.value === 'asc' ? 1 : -1
  return [...filteredIndices.value].sort((a, b) => {
    const av = (a as any)[key]
    const bv = (b as any)[key]
    if (av === bv) return 0
    return (av > bv ? 1 : -1) * dir
  })
})

const pageCount = computed(() => Math.max(Math.ceil(sortedIndices.value.length / pageSize.value), 1))
const pagedIndices = computed(() => sortedIndices.value.slice(page.value * pageSize.value, (page.value + 1) * pageSize.value))

const filteredNodes = computed(() => {
  let nodes = data.value?.nodes || []
  if (nodeFilter.value) {
    const needle = nodeFilter.value.toLowerCase()
    nodes = nodes.filter((n) => `${n.name} ${n.host} ${n.ip}`.toLowerCase().includes(needle))
  }
  return nodes.filter((n) => {
    const hasRole = n.master || n.data || n.ingest || n.coordinating
    if (!hasRole) {
      return roleFilter.value.M && roleFilter.value.D && roleFilter.value.I && roleFilter.value.C
    }
    return (n.master && roleFilter.value.M)
      || (n.data && roleFilter.value.D)
      || (n.ingest && roleFilter.value.I)
      || (n.coordinating && roleFilter.value.C)
  })
})

const allocationMenu = computed(() => data.value?.shard_allocation
  ? [
      { label: 'Disable: none', onSelect: () => disableAllocation('none') },
      { label: 'Disable: primaries', onSelect: () => disableAllocation('primaries') },
      { label: 'Disable: new_primaries', onSelect: () => disableAllocation('new_primaries') },
    ]
  : [{ label: 'Enable allocation', onSelect: () => enableAllocation() }])

const bulkMenu = computed(() => [
  { label: 'Close selected', onSelect: () => confirmBulk('Close selected indices?', 'Close', '/overview/close_indices', 'Closed') },
  { label: 'Open selected', onSelect: () => confirmBulk('Open selected indices?', 'Open', '/overview/open_indices', 'Opened') },
  { label: 'Force merge selected', onSelect: () => confirmBulk('Force merge selected indices?', 'Force merge', '/overview/force_merge', 'Force merged') },
  { label: 'Refresh selected', onSelect: () => confirmBulk('Refresh selected indices?', 'Refresh', '/overview/refresh_indices', 'Refreshed') },
  { label: 'Flush selected', onSelect: () => confirmBulk('Flush selected indices?', 'Flush', '/overview/flush_indices', 'Flushed') },
  { label: 'Clear cache selected', onSelect: () => confirmBulk('Clear cache for selected indices?', 'Clear cache', '/overview/clear_indices_cache', 'Cleared cache for') },
  { label: 'Delete selected', onSelect: () => confirmBulk('Delete selected indices? This cannot be undone.', 'Delete', '/overview/delete_indices', 'Deleted') },
])

function indexMenu(index: OverviewIndex) {
  const closed = index.closed
  return [
    [
      { label: 'Show settings', onSelect: () => inspect('Settings', '/commons/get_index_settings', index.name) },
      { label: 'Show mappings', onSelect: () => inspect('Mappings', '/commons/get_index_mapping', index.name) },
      { label: 'Show stats', onSelect: () => inspect('Stats', '/commons/get_index_stats', index.name) },
      { label: 'Edit index settings', onSelect: () => navigateTo(`/index-settings?index=${encodeURIComponent(index.name)}`) },
    ],
    [
      {
        label: closed ? 'Open' : 'Close',
        onSelect: () => closed
          ? runIndexAction('/overview/open_indices', index.name, 'Opened index')
          : confirmIndexAction('Close index?', 'Close', '/overview/close_indices', index.name, 'Closed index'),
      },
      {
        label: 'Force merge',
        disabled: closed,
        onSelect: () => confirmIndexAction('Force merge index?', 'Force merge', '/overview/force_merge', index.name, 'Force merged index'),
      },
      { label: 'Refresh', disabled: closed, onSelect: () => runIndexAction('/overview/refresh_indices', index.name, 'Refreshed index') },
      { label: 'Flush', disabled: closed, onSelect: () => runIndexAction('/overview/flush_indices', index.name, 'Flushed index') },
      { label: 'Clear cache', disabled: closed, onSelect: () => runIndexAction('/overview/clear_indices_cache', index.name, 'Cleared cache for') },
    ],
    [{
      label: 'Delete',
      color: 'error',
      onSelect: () => confirmIndexAction('Delete index? This cannot be undone.', 'Delete', '/overview/delete_indices', index.name, 'Deleted index'),
    }],
  ]
}

function indexHeaderClass(index: OverviewIndex) {
  if (index.status === 'red') return 'border-t-2 border-t-rose-500 bg-rose-500/5'
  if (index.status === 'yellow') return 'border-t-2 border-t-amber-500 bg-amber-500/5'
  return ''
}

function focusUnhealthy() {
  unhealthyOnly.value = true
  indexSortKey.value = 'unhealthy'
  indexSortDir.value = 'desc'
  page.value = 0
  if (unhealthyIndices.value.some((i) => i.special)) {
    showSpecial.value = true
  }
}

function filterByDataStream(name?: string) {
  if (!name) return
  dataStreamFilter.value = name
  page.value = 0
}

function shardMenu(index: OverviewIndex, shard: OverviewShard) {
  return [
    { label: 'Display shard stats', onSelect: () => inspectShard(index, shard) },
    {
      label: relocatingShard.value?.index === index.name && relocatingShard.value?.shard.shard === shard.shard
        ? 'Unselect for relocation'
        : 'Select for relocation',
      onSelect: () => toggleRelocation(index, shard),
    },
  ]
}

function shardClass(shard: OverviewShard) {
  switch (shard.state) {
    case 'STARTED': return 'shard-badge--started'
    case 'INITIALIZING': return 'shard-badge--initializing'
    case 'RELOCATING': return 'shard-badge--relocating'
    case 'RECOVERING': return 'shard-badge--recovering'
    default: return 'shard-badge--unassigned'
  }
}

function toggleRelocation(index: OverviewIndex, shard: OverviewShard) {
  if (relocatingShard.value?.index === index.name && relocatingShard.value?.shard.shard === shard.shard) {
    relocatingShard.value = null
  } else {
    relocatingShard.value = { index: index.name, shard }
  }
}

function canRelocateTo(index: OverviewIndex, node: OverviewNode) {
  const selected = relocatingShard.value
  if (!selected || selected.index !== index.name) return false
  if (selected.shard.node === node.id) return false
  return !(index.shards?.[node.id] || []).some((s) => s.shard === selected.shard.shard)
}

async function relocate(index: OverviewIndex, node: OverviewNode) {
  const selected = relocatingShard.value
  if (!selected) return
  await apiCall('POST', '/overview/relocate_shard', {
    host: clusterHost.value,
    index: index.name,
    shard: selected.shard.shard,
    from: selected.shard.node,
    to: node.id,
  })
  relocatingShard.value = null
  await load()
}

async function inspectShard(index: OverviewIndex, shard: OverviewShard) {
  const resp = await apiCall('POST', '/overview/get_shard_stats', {
    host: clusterHost.value,
    index: index.name,
    shard: shard.shard,
    node: shard.node || 'unassigned',
    primary: shard.primary,
  })
  inspectorTitle.value = `Shard ${shard.shard}: ${index.name}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

async function inspectNode(node: OverviewNode) {
  const resp = await apiCall('POST', '/commons/get_node_stats', { host: clusterHost.value, node: node.id })
  inspectorTitle.value = `Node stats: ${node.name}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

async function inspect(title: string, url: string, index: string) {
  const resp = await apiCall('POST', url, { host: clusterHost.value, index })
  inspectorTitle.value = `${title}: ${index}`
  inspectorData.value = resp?.body ?? resp
  inspectorOpen.value = true
}

function reportAction(resp: any, successLabel: string, target: string) {
  if (resp?.status === 200) {
    const failed = resp?.body?._shards?.failed
    if (typeof failed === 'number' && failed > 0) {
      pushAlert(`${successLabel} ${target} with ${failed} failed shard(s).`, 'warning')
    } else {
      pushAlert(`${successLabel} ${target}`, 'success')
    }
    return
  }
  const detail = extractEnvelopeError(resp?.body)
  pushAlert(
    detail
      ? `${successLabel} ${target} failed (${resp?.status || 'network error'}): ${detail}`
      : `${successLabel} ${target} failed (${resp?.status || 'network error'}).`,
    'danger',
  )
}

async function runIndexAction(url: string, index: string, successLabel: string) {
  const resp = await apiCall('POST', url, { host: clusterHost.value, indices: index }, { suppressErrorAlert: true })
  reportAction(resp, successLabel, index)
  await load()
}

function confirmIndexAction(title: string, label: string, url: string, index: string, successLabel: string) {
  confirmTitle.value = title
  confirmDescription.value = `This will affect index ${index}.`
  confirmLabel.value = label
  pendingAction.value = async () => {
    await runIndexAction(url, index, successLabel)
  }
  confirmOpen.value = true
}

function confirmBulk(title: string, label: string, url: string, successLabel: string) {
  const names = filteredIndices.value.map((i) => i.name)
  if (!names.length) {
    pushAlert('No indices selected.', 'warning')
    return
  }
  confirmTitle.value = title
  confirmDescription.value = `This will affect ${names.length} filtered indices.`
  confirmLabel.value = label
  pendingAction.value = async () => {
    const resp = await apiCall('POST', url, { host: clusterHost.value, indices: names.join(',') }, { suppressErrorAlert: true })
    reportAction(resp, successLabel, `${names.length} indices`)
    await load()
  }
  confirmOpen.value = true
}

async function runConfirmed() {
  confirmOpen.value = false
  if (pendingAction.value) await pendingAction.value()
  pendingAction.value = null
}

async function disableAllocation(kind: string) {
  await apiCall('POST', '/overview/disable_shard_allocation', { host: clusterHost.value, kind })
  await load()
}

async function enableAllocation() {
  await apiCall('POST', '/overview/enable_shard_allocation', { host: clusterHost.value })
  await load()
}

async function load() {
  if (!clusterHost.value) {
    error.value = 'No active cluster connection. Connect to a cluster first.'
    loading.value = false
    return
  }
  loading.value = true
  const resp = await apiCall('POST', '/overview', { host: clusterHost.value }, { suppressErrorAlert: true })
  if (resp?.status === 200) {
    data.value = resp.body
    error.value = ''
  } else if (resp?.status === 401 || resp?.status === 403) {
    error.value = `Unauthorized (${resp.status}): cluster rejected the request. Reconnect with valid credentials.`
  } else {
    const detail = extractEnvelopeError(resp?.body)
    error.value = detail
      ? `Failed to load cluster overview (${resp?.status || 'network error'}): ${detail}`
      : `Failed to load cluster overview (${resp?.status || 'network error'}).`
  }
  loading.value = false
}

function applyAutoRefresh() {
  if (timer) clearInterval(timer)
  timer = null
  if (autoRefresh.value !== 'off') {
    timer = setInterval(load, Number(autoRefresh.value) * 1000)
  }
}

watch(autoRefresh, applyAutoRefresh)
watch([indexFilter, showClosed, showSpecial, showBacking, unhealthyOnly, dataStreamFilter, pageSize], () => { page.value = 0 })
watch(pageCount, (count) => { if (page.value >= count) page.value = Math.max(count - 1, 0) })

onMounted(() => {
  if (typeof window !== 'undefined') {
    pageSize.value = Math.max(Math.floor(window.innerWidth / 260), 3)
  }
  load()
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>
