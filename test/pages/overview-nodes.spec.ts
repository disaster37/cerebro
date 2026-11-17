import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import OverviewPage from '@/pages/overview.vue'
import NodesPage from '@/pages/nodes/index.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: 'admin',
    password: 'secret',
    trustCertificate: false,
  }
  ;(globalThis as any).useState('clusterName').value = 'prod-cluster'
}

function mountPage(page: any) {
  return mount(page)
}

describe('overview and nodes pages', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('overview sends clusterHost with credentials even when clusterName differs', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: {
        number_of_nodes: 3,
        active_shards: 5,
        unassigned_shards: 0,
        total_indices: 2,
        indices: [{ name: 'idx', doc_count: 1, total_size_in_bytes: 1024 }],
      },
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/overview', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'admin', password: 'secret' },
    }))
    expect(wrapper.find('[data-testid="overview-error"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('idx')
  })

  it('nodes sends clusterHost with credentials even when clusterName differs', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: [{ id: 'n1', name: 'node-1', cpu_percent: 10 }],
    })

    const wrapper = mountPage(NodesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/nodes', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'admin', password: 'secret' },
    }))
    expect(wrapper.find('[data-testid="nodes-error"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('node-1')
  })

  it('overview renders an error card and alert on 401', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 401,
      body: { error: 'unauthorized' },
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect(wrapper.find('[data-testid="overview-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Connect to Cluster')
    expect((globalThis as any).useState('alerts').value).toHaveLength(1)
  })

  it('nodes renders an error card and alert on 502', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 502,
      body: { error: 'bad gateway' },
    })

    const wrapper = mountPage(NodesPage)
    await flushPromises()

    expect(wrapper.find('[data-testid="nodes-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('bad gateway')
    expect((globalThis as any).useState('alerts').value).toHaveLength(1)
  })

  it('overview renders the shard allocation matrix and node resource bars', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: {
        cluster_name: 'prod',
        status: 'green',
        number_of_nodes: 1,
        active_shards: 1,
        unassigned_shards: 0,
        relocating_shards: 0,
        initializing_shards: 0,
        docs_count: 10,
        size_in_bytes: 1024,
        total_indices: 1,
        closed_indices: 0,
        special_indices: 0,
        shard_allocation: true,
        indices: [{
          name: 'idx', closed: false, special: false, unhealthy: false, doc_count: 10, deleted_docs: 0,
          size_in_bytes: 1024, total_size_in_bytes: 1024, aliases: ['a1'], num_shards: 1, num_replicas: 0,
          shards: { n1: [{ index: 'idx', shard: 0, primary: true, state: 'STARTED', node: 'n1' }] },
        }],
        nodes: [{
          id: 'n1', name: 'node-1', host: 'h1', ip: '1.1.1.1', current_master: true, master: true,
          data: true, coordinating: false, ingest: false, es_version: '9.4.3', jvm_version: '21',
          load_average: 0.5, available_processors: 4, cpu_percent: 12,
          heap: { used: 10, committed: 20, used_percent: 10, max: 100 },
          disk: { total: 100, free: 40, used_percent: 60 }, attributes: { zone: 'a' },
        }],
      },
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect(wrapper.findAll('.shard-badge').length).toBeGreaterThan(0)
    expect(wrapper.text()).toContain('node-1')
    expect(wrapper.text()).toContain('zone=a')
    expect(wrapper.findAll('.uprogress').length).toBeGreaterThan(0)
  })

  it('nodes renders OS CPU, process CPU and N/A load', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: [{
        id: 'n1', name: 'node-1', host: 'h1', ip: '1.1.1.1', current_master: true, master: true,
        data: true, coordinating: false, ingest: false, es_version: '9.4.3', jvm_version: '21',
        load_average: null, load_average_5m: null, load_average_15m: null, available_processors: 4,
        cpu_percent: 42, process_cpu_percent: 7, uptime_in_millis: 1000,
        heap: { used: 10, committed: 20, used_percent: 10, max: 100 },
        disk: { total: 100, free: 40, used_percent: 60 }, attributes: {},
      }],
    })

    const wrapper = mountPage(NodesPage)
    await flushPromises()

    expect(wrapper.text()).toContain('42%')
    expect(wrapper.text()).toContain('process: 7%')
    expect(wrapper.text()).toContain('N/A')
  })

  it('overview renders a disconnected prompt without fetching on reload', async () => {
    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="overview-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No active cluster connection')
  })

  it('nodes renders a disconnected prompt without fetching on reload', async () => {
    const wrapper = mountPage(NodesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="nodes-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No active cluster connection')
  })
})
