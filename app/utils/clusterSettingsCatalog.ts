export interface ClusterSettingDef {
  key: string
  label: string
  description?: string
  type: 'boolean' | 'number' | 'string' | 'enum'
  options?: string[]
}

export interface ClusterSettingGroup {
  title: string
  settings: ClusterSettingDef[]
}

export const STATIC_PREFIXES = [
  'node.',
  'path.',
  'network.',
  'http.',
  'transport.',
  'cluster.name',
  'discovery.seed_hosts',
]

export const ALLOWED_DYNAMIC_PREFIXES = [
  'logger.',
  'cluster.routing.allocation.include.',
  'cluster.routing.allocation.exclude.',
  'cluster.routing.allocation.require.',
  'cluster.routing.allocation.awareness.force.',
]

export const CLUSTER_SETTINGS_CATALOG: ClusterSettingGroup[] = [
  {
    title: 'Cluster Blocks',
    settings: [
      { key: 'cluster.blocks.read_only', label: 'Read only', type: 'boolean' },
      { key: 'cluster.blocks.read_only_allow_delete', label: 'Read only (allow delete)', type: 'boolean' },
      { key: 'cluster.routing.allocation.enable', label: 'Shard allocation', type: 'enum', options: ['all', 'primaries', 'new_primaries', 'none'] },
    ],
  },
  {
    title: 'Shard Allocation & Disk Watermarks',
    settings: [
      { key: 'cluster.routing.allocation.disk.threshold_enabled', label: 'Disk threshold enabled', type: 'boolean' },
      { key: 'cluster.routing.allocation.disk.watermark.low', label: 'Low watermark', type: 'string' },
      { key: 'cluster.routing.allocation.disk.watermark.high', label: 'High watermark', type: 'string' },
      { key: 'cluster.routing.allocation.disk.watermark.flood_stage', label: 'Flood stage watermark', type: 'string' },
      { key: 'cluster.routing.allocation.node_initial_primaries_recoveries', label: 'Initial primaries recoveries', type: 'number' },
      { key: 'cluster.routing.allocation.node_concurrent_recoveries', label: 'Concurrent recoveries', type: 'number' },
      { key: 'cluster.routing.allocation.cluster_concurrent_rebalance', label: 'Concurrent rebalance', type: 'number' },
      { key: 'cluster.routing.allocation.allow_rebalance', label: 'Allow rebalance', type: 'enum', options: ['always', 'indices_primaries_active', 'indices_all_active'] },
      { key: 'cluster.routing.allocation.balance.shard', label: 'Balance: shard', type: 'number' },
      { key: 'cluster.routing.allocation.balance.index', label: 'Balance: index', type: 'number' },
      { key: 'cluster.routing.allocation.balance.primary', label: 'Balance: primary', type: 'number' },
      { key: 'cluster.routing.allocation.awareness.attributes', label: 'Awareness attributes', type: 'string' },
    ],
  },
  {
    title: 'Shard Rebalancing & Recovery',
    settings: [
      { key: 'cluster.routing.rebalance.enable', label: 'Rebalance enable', type: 'enum', options: ['all', 'primaries', 'replicas', 'none'] },
      { key: 'cluster.routing.allocation.total_shards_per_node', label: 'Total shards per node', type: 'number' },
      { key: 'cluster.max_shards_per_node', label: 'Max shards per node', type: 'number' },
      { key: 'cluster.max_shards_per_node.frozen', label: 'Max shards per node (frozen)', type: 'number' },
      { key: 'indices.recovery.max_bytes_per_sec', label: 'Recovery max bytes/sec', type: 'string' },
      { key: 'indices.recovery.max_concurrent_file_chunks', label: 'Recovery concurrent file chunks', type: 'number' },
      { key: 'indices.recovery.max_concurrent_operations', label: 'Recovery concurrent operations', type: 'number' },
      { key: 'indices.recovery.max_concurrent_snapshot_file_chunks', label: 'Recovery concurrent snapshot chunks', type: 'number' },
    ],
  },
  {
    title: 'Circuit Breakers',
    settings: [
      { key: 'indices.breaker.total.limit', label: 'Total breaker limit', type: 'string' },
      { key: 'indices.breaker.total.use_real_memory', label: 'Total breaker uses real memory', type: 'boolean' },
      { key: 'indices.breaker.fielddata.limit', label: 'Fielddata breaker limit', type: 'string' },
      { key: 'indices.breaker.fielddata.overhead', label: 'Fielddata breaker overhead', type: 'number' },
      { key: 'indices.breaker.request.limit', label: 'Request breaker limit', type: 'string' },
      { key: 'indices.breaker.request.overhead', label: 'Request breaker overhead', type: 'number' },
      { key: 'indices.breaker.accounting.limit', label: 'Accounting breaker limit', type: 'string' },
      { key: 'network.breaker.inflight_requests.limit', label: 'Inflight requests breaker limit', type: 'string' },
      { key: 'script.max_compilations_rate', label: 'Max script compilations rate', type: 'string' },
      { key: 'script.context.search.max_compilations_rate', label: 'Search context compilations rate', type: 'string' },
    ],
  },
  {
    title: 'Search & Action Limits',
    settings: [
      { key: 'action.destructive_requires_name', label: 'Destructive actions require name', type: 'boolean' },
      { key: 'action.auto_create_index', label: 'Auto create index', type: 'string' },
      { key: 'search.max_open_scroll_context', label: 'Max open scroll contexts', type: 'number' },
      { key: 'search.max_buckets', label: 'Max buckets', type: 'number' },
      { key: 'search.default_search_timeout', label: 'Default search timeout', type: 'string' },
      { key: 'search.allow_expensive_queries', label: 'Allow expensive queries', type: 'boolean' },
      { key: 'search.max_async_search_response_size', label: 'Max async search response size', type: 'string' },
      { key: 'indices.query.bool.max_clause_count', label: 'Max bool clause count', type: 'number' },
    ],
  },
  {
    title: 'Indexing & Scripting',
    settings: [
      { key: 'indices.memory.index_buffer_size', label: 'Index buffer size', type: 'string' },
      { key: 'indices.memory.min_index_buffer_size', label: 'Min index buffer size', type: 'string' },
      { key: 'indices.memory.max_index_buffer_size', label: 'Max index buffer size', type: 'string' },
      { key: 'indices.fielddata.cache.size', label: 'Fielddata cache size', type: 'string' },
      { key: 'indices.queries.cache.size', label: 'Queries cache size', type: 'string' },
      { key: 'indices.requests.cache.size', label: 'Requests cache size', type: 'string' },
      { key: 'indices.requests.cache.expire', label: 'Requests cache expire', type: 'string' },
      { key: 'script.allowed_types', label: 'Allowed script types', type: 'string' },
      { key: 'script.allowed_contexts', label: 'Allowed script contexts', type: 'string' },
      { key: 'script.max_size_in_bytes', label: 'Max script size', type: 'number' },
      { key: 'thread_pool.write.queue_size', label: 'Write queue size', type: 'number' },
      { key: 'thread_pool.search.queue_size', label: 'Search queue size', type: 'number' },
    ],
  },
  {
    title: 'Lifecycle / ISM',
    settings: [
      { key: 'indices.lifecycle.poll_interval', label: 'ILM poll interval', type: 'string' },
      { key: 'indices.lifecycle.history_index_enabled', label: 'ILM history index enabled', type: 'boolean' },
      { key: 'plugins.index_state_management.job_interval', label: 'ISM job interval', type: 'number' },
      { key: 'plugins.index_state_management.enabled', label: 'ISM enabled', type: 'boolean' },
      { key: 'plugins.index_state_management.history.enabled', label: 'ISM history enabled', type: 'boolean' },
      { key: 'plugins.index_state_management.coordinator.sweep_period', label: 'ISM coordinator sweep period', type: 'string' },
      { key: 'plugins.index_state_management.rollover_alias', label: 'ISM rollover alias', type: 'string' },
      { key: 'xpack.ml.max_machine_memory_percent', label: 'ML max machine memory percent', type: 'number' },
      { key: 'xpack.monitoring.collection.enabled', label: 'Monitoring collection enabled', type: 'boolean' },
      { key: 'xpack.security.audit.enabled', label: 'Security audit enabled', type: 'boolean' },
    ],
  },
]

export function isStaticSetting(key: string): boolean {
  return STATIC_PREFIXES.some((prefix) => key === prefix || key.startsWith(prefix))
}

export function hasAllowedDynamicPrefix(key: string): boolean {
  return ALLOWED_DYNAMIC_PREFIXES.some((prefix) => key.startsWith(prefix))
}

export interface SettingKeyValidation {
  valid: boolean
  static: boolean
  message?: string
}

export function validateSettingKey(key: string, knownKeys: Set<string>): SettingKeyValidation {
  const trimmed = key.trim()
  if (!trimmed) return { valid: false, static: false, message: 'Setting key is required' }
  const known = knownKeys.has(trimmed) || hasAllowedDynamicPrefix(trimmed)
  if (!known) {
    return { valid: false, static: false, message: `Parameter '${trimmed}' does not exist on this cluster` }
  }
  if (isStaticSetting(trimmed)) {
    return {
      valid: true,
      static: true,
      message: 'This parameter exists in cluster defaults but appears to be a static node setting (requires elasticsearch.yml / opensearch.yml and restart)',
    }
  }
  return { valid: true, static: false }
}
