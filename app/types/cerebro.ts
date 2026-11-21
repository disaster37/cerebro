export interface UnassignedShardInfo {
  reason: string
  at?: string
  details?: string
  last_allocation_status?: string
}

export interface OverviewShard {
  index: string
  shard: number
  primary: boolean
  state: string
  node: string | null
  relocating_node?: string | null
  unassigned_info?: UnassignedShardInfo
}

export interface OverviewIndex {
  name: string
  closed: boolean
  special: boolean
  backing_index: boolean
  data_stream?: string
  is_write_index?: boolean
  status: 'green' | 'yellow' | 'red' | 'closed'
  unhealthy: boolean
  unreplicated: boolean
  unassigned_shard_count: number
  initializing_shard_count: number
  relocating_shard_count: number
  doc_count: number
  deleted_docs: number
  size_in_bytes: number
  total_size_in_bytes: number
  aliases: string[]
  num_shards: number
  num_replicas: number
  shards: Record<string, OverviewShard[]>
}

export interface DataStreamSummary {
  name: string
  status: string
  generation: number
  template: string
  timestamp_field: string
  write_index: string
  backing_indices: string[]
  doc_count: number
  size_in_bytes: number
  total_size_in_bytes: number
  hidden?: boolean
  system?: boolean
}

export interface TemplateEntry {
  name: string
  type: 'composable' | 'legacy'
  patterns: string[]
  order?: number
  version?: number
  data_stream?: boolean
  template: any
}

export type NodeCpuSource = 'resource_tracker' | 'os' | 'process'

export interface OverviewNode {
  id: string
  name: string
  host: string
  ip: string
  current_master: boolean
  master: boolean
  data: boolean
  coordinating: boolean
  ingest: boolean
  es_version: string
  jvm_version: string
  load_average: number | null
  available_processors: number
  cpu_percent: number | null
  cpu_source?: NodeCpuSource
  resource_cpu_percent?: number | null
  os_cpu_percent?: number | null
  process_cpu_percent: number | null
  heap: { used: number; committed: number; used_percent: number; max: number }
  disk: { total: number; free: number; used_percent: number }
  attributes: Record<string, string>
}

export interface OverviewData {
  cluster_name: string
  status: string
  number_of_nodes: number
  active_primary_shards: number
  active_shards: number
  relocating_shards: number
  initializing_shards: number
  unassigned_shards: number
  docs_count: number
  size_in_bytes: number
  total_indices: number
  closed_indices: number
  special_indices: number
  indices: OverviewIndex[]
  data_streams: DataStreamSummary[]
  nodes: OverviewNode[]
  shard_allocation: boolean
}

export interface NodeEntry {
  id: string
  name: string
  host: string
  ip: string
  current_master: boolean
  master: boolean
  data: boolean
  coordinating: boolean
  ingest: boolean
  es_version: string
  jvm_version: string
  load_average: number | null
  load_average_5m?: number | null
  load_average_15m?: number | null
  available_processors: number
  cpu_percent: number | null
  cpu_source?: NodeCpuSource
  resource_cpu_percent?: number | null
  os_cpu_percent?: number | null
  process_cpu_percent: number | null
  uptime_in_millis: number
  attributes: Record<string, string>
  heap: { used: number; committed: number; used_percent: number; max: number }
  disk: { total: number; free: number; used_percent: number }
}

export interface SnapshotFailure {
  index: string
  shard_id: number
  node_id: string
  status: string
  reason: string
}

export interface SnapshotEntry {
  snapshot: string
  uuid: string
  version: string
  indices: string[]
  data_streams: string[]
  include_global_state: boolean
  state: string
  start_time_in_millis: number
  end_time_in_millis: number
  duration_in_millis: number
  failures: SnapshotFailure[]
  shards: { total: number; failed: number; successful: number }
}

export interface RepositoryEntry {
  name: string
  type: string
  settings: Record<string, any>
}

export interface VerifiedRepoNode {
  id: string
  name: string
}

export interface RepositoryVerifyStatus {
  name: string
  verified: boolean
  status: 'CONNECTED' | 'IN_PROGRESS' | 'PARTIAL_FAILURES' | 'ERROR'
  node_count: number
  nodes: VerifiedRepoNode[]
  snapshot_count: number
  active_count: number
  failed_count: number
  size_computed: boolean
  size_in_bytes: number | null
  logical_size_in_bytes: number | null
  size_skipped_reason?: string
  error?: string
}

export interface ClusterSettingsResponse {
  defaults?: Record<string, any>
  persistent?: Record<string, any>
  transient?: Record<string, any>
}

export interface RestMetadata {
  indices: string[]
  fieldsByIndex: Record<string, string[]>
  allFields: string[]
  backendType: string
  host: string
}

export interface IndexPolicySummary {
  id: string
  version: number
  modified_date: string
  phases: string[]
  managed_count: number
  failed_count: number
  raw_policy: any
}

export interface ManagedIndexStatus {
  index: string
  policy_id: string
  phase: string
  action: string
  step: string
  failed: boolean
  failed_step?: string
  retry_count: number
  reason?: string
  age?: string
  available_states?: string[]
}

export interface IndexLifecyclePayload {
  backendType: string
  featureName: string
  operationMode: string
  policies: IndexPolicySummary[]
  managedIndices: ManagedIndexStatus[]
  unsupported?: boolean
  warning?: string
}

export interface SnapshotPolicySummary {
  id: string
  name: string
  repository: string
  schedule: string
  indices: string[]
  enabled: boolean
  current_state: string
  last_success_name?: string
  last_success_time?: number
  last_failure_name?: string
  last_failure_time?: number
  last_failure_message?: string
  next_execution_millis?: number
  snapshots_taken: number
  snapshots_failed: number
  raw_policy: any
}

export interface SnapshotLifecyclePayload {
  backendType?: string
  featureName?: string
  policies: SnapshotPolicySummary[]
  unsupported?: boolean
  warning?: string
}
