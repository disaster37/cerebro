import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import OverviewPage from '@/pages/overview.vue'
import NodesPage from '@/pages/nodes/index.vue'
import NodeResourceBars from '@/components/overview/NodeResourceBars.vue'
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
    // Initial load suppresses the global alert because the page renders an
    // inline error card.
    expect((globalThis as any).useState('alerts').value).toHaveLength(0)
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
    expect((globalThis as any).useState('alerts').value).toHaveLength(0)
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

describe('overview shard health', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  function baseBody(indices: any[], nodes: any[] = []) {
    return {
      cluster_name: 'prod',
      status: 'yellow',
      number_of_nodes: nodes.length,
      active_shards: 1,
      unassigned_shards: 2,
      relocating_shards: 0,
      initializing_shards: 0,
      docs_count: 10,
      size_in_bytes: 1024,
      total_indices: indices.length,
      closed_indices: 0,
      special_indices: indices.filter((i) => i.special).length,
      shard_allocation: true,
      indices,
      data_streams: [],
      nodes,
    }
  }

  it('renders per-index health, unassigned primary/replica badges and 0 replicas', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: baseBody([{
        name: 'idx', closed: false, special: false, backing_index: false, status: 'yellow', unhealthy: true,
        unreplicated: true, unassigned_shard_count: 1, initializing_shard_count: 0, relocating_shard_count: 0,
        doc_count: 10, deleted_docs: 0, size_in_bytes: 1024, total_size_in_bytes: 1024, aliases: [],
        num_shards: 1, num_replicas: 0,
        shards: {
          unassigned: [
            { index: 'idx', shard: 0, primary: false, state: 'UNASSIGNED', node: null, unassigned_info: { reason: 'NO_REPLICA' } },
            { index: 'idx', shard: 0, primary: true, state: 'UNASSIGNED', node: null, unassigned_info: { reason: 'INDEX_CREATED' } },
          ],
        },
      }]),
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect(wrapper.text()).toContain('yellow (1 unassigned)')
    expect(wrapper.text()).toContain('0 replicas')
    expect(wrapper.find('[data-testid="unhealthy-banner"]').exists()).toBe(true)
    expect(wrapper.findAll('.shard-badge--unassigned-primary').length).toBe(1)
    expect(wrapper.findAll('.shard-badge--unassigned').length).toBe(1)
    const titles = wrapper.findAll('.shard-badge').map((b) => b.attributes('title'))
    expect(titles.some((t) => t?.includes('PRIMARY') && t?.includes('INDEX_CREATED'))).toBe(true)
    expect(titles.some((t) => t?.includes('REPLICA') && t?.includes('NO_REPLICA'))).toBe(true)
  })

  it('shows data stream backing indices by default with DS and write badges', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: baseBody([{
        name: '.ds-logs-app-000001', closed: false, special: false, backing_index: true,
        data_stream: 'logs-app', is_write_index: true, status: 'green', unhealthy: false,
        unreplicated: false, unassigned_shard_count: 0, initializing_shard_count: 0, relocating_shard_count: 0,
        doc_count: 10, deleted_docs: 0, size_in_bytes: 1024, total_size_in_bytes: 1024, aliases: [],
        num_shards: 1, num_replicas: 0,
        shards: { n1: [{ index: '.ds-logs-app-000001', shard: 0, primary: true, state: 'STARTED', node: 'n1' }] },
      }], [{
        id: 'n1', name: 'node-1', host: 'h1', ip: '1.1.1.1', current_master: true, master: true,
        data: true, coordinating: false, ingest: false, es_version: '9.4.3', jvm_version: '21',
        load_average: 0.5, available_processors: 4, cpu_percent: 12,
        heap: { used: 10, committed: 20, used_percent: 10, max: 100 },
        disk: { total: 100, free: 40, used_percent: 60 }, attributes: {},
      }]),
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    expect(wrapper.text()).toContain('.ds-logs-app-000001')
    expect(wrapper.text()).toContain('DS: logs-app')
    expect(wrapper.text()).toContain('write')
  })

  it('focuses unhealthy indices from the warning banner', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: baseBody([{
        name: 'idx', closed: false, special: false, backing_index: false, status: 'yellow', unhealthy: true,
        unreplicated: false, unassigned_shard_count: 1, initializing_shard_count: 0, relocating_shard_count: 0,
        doc_count: 10, deleted_docs: 0, size_in_bytes: 1024, total_size_in_bytes: 1024, aliases: [],
        num_shards: 1, num_replicas: 1,
        shards: { unassigned: [{ index: 'idx', shard: 0, primary: false, state: 'UNASSIGNED', node: null }] },
      }]),
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    const focus = wrapper.findAll('button').find((b) => b.text().includes('Focus unhealthy indices'))
    expect(focus).toBeTruthy()
    await focus!.trigger('click')
    await flushPromises()

    const unhealthy = wrapper.findAll('input[type="checkbox"]').find((el) => el.element.parentElement?.textContent?.includes('unhealthy only'))
    expect(unhealthy).toBeTruthy()
    expect((unhealthy!.element as HTMLInputElement).checked).toBe(true)
  })

  it('keeps multi-role nodes visible when one role filter is disabled', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: baseBody([], [{
        id: 'n1', name: 'node-1', host: 'h1', ip: '1.1.1.1', current_master: true, master: true,
        data: true, coordinating: false, ingest: false, es_version: '9.4.3', jvm_version: '21',
        load_average: 0.5, available_processors: 4, cpu_percent: 12,
        heap: { used: 10, committed: 20, used_percent: 10, max: 100 },
        disk: { total: 100, free: 40, used_percent: 60 }, attributes: {},
      }]),
    })

    const wrapper = mountPage(OverviewPage)
    await flushPromises()

    const masterToggle = wrapper.findAll('button').find((b) => b.text() === 'M')
    expect(masterToggle).toBeTruthy()
    await masterToggle!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('node-1')
  })
})

function nodeEntry(overrides: Record<string, any> = {}) {
  return {
    id: 'n1', name: 'node-1', host: 'h1', ip: '1.1.1.1', current_master: true, master: true,
    data: true, coordinating: false, ingest: false, es_version: '9.4.3', jvm_version: '21',
    load_average: 0.5, load_average_5m: 0.4, load_average_15m: 0.3, available_processors: 4,
    cpu_percent: 12.34, cpu_source: 'resource_tracker', resource_cpu_percent: 12.34,
    os_cpu_percent: null, process_cpu_percent: null, uptime_in_millis: 1000,
    heap: { used: 10, committed: 20, used_percent: 10, max: 100 },
    disk: { total: 100, free: 40, used_percent: 60 }, attributes: {},
    ...overrides,
  }
}

async function mountNodes(nodes: any[]) {
  setConnection()
  ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: nodes })
  const wrapper = mountPage(NodesPage)
  await flushPromises()
  return wrapper
}

describe('nodes CPU formatting and source hints', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('formats resource tracker float CPU and labels its source', async () => {
    const wrapper = await mountNodes([nodeEntry()])
    expect(wrapper.text()).toContain('12.3%')
    expect(wrapper.text()).toContain('process CPU (resource tracker)')
  })

  it('preserves 0% idle and shows process N/A', async () => {
    const wrapper = await mountNodes([nodeEntry({
      cpu_percent: 0, cpu_source: 'os', resource_cpu_percent: null, os_cpu_percent: 0, process_cpu_percent: null,
    })])
    expect(wrapper.text()).toContain('0%')
    expect(wrapper.text()).toContain('process: N/A')
  })

  it('formats integer OS CPU and process fallback', async () => {
    const wrapper = await mountNodes([nodeEntry({
      cpu_percent: 42, cpu_source: 'os', resource_cpu_percent: null, os_cpu_percent: 42, process_cpu_percent: 7,
    })])
    expect(wrapper.text()).toContain('42%')
    expect(wrapper.text()).toContain('process: 7%')
  })

  it('shows N/A when every CPU source is unavailable', async () => {
    const wrapper = await mountNodes([nodeEntry({
      cpu_percent: null, cpu_source: undefined, resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: null,
    })])
    expect(wrapper.text()).toContain('N/A')
    expect(wrapper.text()).toContain('process: N/A')
  })

  it('labels process MXBean fallback as OS N/A', async () => {
    const wrapper = await mountNodes([nodeEntry({
      cpu_percent: 7, cpu_source: 'process', resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: 7,
    })])
    expect(wrapper.text()).toContain('process CPU (OS: N/A)')
  })

  it('sorts unavailable CPU last in both directions', async () => {
    const wrapper = await mountNodes([
      nodeEntry({ id: 'n1', name: 'node-a', cpu_percent: 12.34 }),
      nodeEntry({ id: 'n2', name: 'node-b', cpu_percent: null, cpu_source: undefined, resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: null }),
      nodeEntry({ id: 'n3', name: 'node-c', cpu_percent: 5, cpu_source: 'os', resource_cpu_percent: null, os_cpu_percent: 5, process_cpu_percent: null }),
    ])
    const header = wrapper.find('[data-testid="sort-cpu"]')
    expect(header.exists()).toBe(true)

    await header.trigger('click')
    let names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('node-c')
    expect(names[1]).toContain('node-a')
    expect(names[2]).toContain('node-b')

    await header.trigger('click')
    names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('node-a')
    expect(names[1]).toContain('node-c')
    expect(names[2]).toContain('node-b')
  })

  it('plots only non-null CPU datasets and enables the legend', async () => {
    const wrapper = await mountNodes([
      nodeEntry({ id: 'n1', name: 'node-a', cpu_percent: 12.34, process_cpu_percent: null }),
      nodeEntry({ id: 'n2', name: 'node-b', cpu_percent: null, cpu_source: undefined, resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: null }),
    ])
    const charts = wrapper.findAll('button').find((b) => b.text().includes('Charts'))
    await charts!.trigger('click')
    await flushPromises()

    const chart = wrapper.find('.chart-bar')
    const data = JSON.parse(chart.attributes('data-chart') || '{}')
    const labels = data.datasets.map((d: any) => d.label)
    expect(labels).toContain('CPU %')
    expect(labels).not.toContain('JVM Process CPU %')
    expect(labels).toContain('Heap %')
    expect(labels).toContain('Disk %')

    const options = JSON.parse(chart.attributes('data-options') || '{}')
    expect(options.plugins.legend.display).toBe(true)
    expect(options.scales.x.min).toBe(0)
  })

  it('omits the CPU dataset when every node CPU is unavailable', async () => {
    const wrapper = await mountNodes([
      nodeEntry({ id: 'n1', name: 'node-a', cpu_percent: null, cpu_source: undefined, resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: null }),
    ])
    const charts = wrapper.findAll('button').find((b) => b.text().includes('Charts'))
    await charts!.trigger('click')
    await flushPromises()

    const data = JSON.parse(wrapper.find('.chart-bar').attributes('data-chart') || '{}')
    expect(data.datasets.map((d: any) => d.label)).not.toContain('CPU %')
  })
})

describe('NodeResourceBars CPU tooltip', () => {
  beforeEach(() => resetTestStubs())

  function tooltipFor(overrides: Record<string, any>) {
    const wrapper = mount(NodeResourceBars, { props: { node: nodeEntry(overrides) } })
    return wrapper.findAll('.utooltip').map((t) => t.attributes('title'))
  }

  it('labels resource tracker CPU', () => {
    expect(tooltipFor({ cpu_percent: 12.34, cpu_source: 'resource_tracker', resource_cpu_percent: 12.34 }))
      .toContain('CPU 12.3% (process CPU — resource tracker)')
  })

  it('labels process CPU', () => {
    expect(tooltipFor({ cpu_percent: 7, cpu_source: 'process', resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: 7 }))
      .toContain('CPU 7% (process CPU)')
  })

  it('labels OS CPU without a source suffix', () => {
    expect(tooltipFor({ cpu_percent: 42, cpu_source: 'os', resource_cpu_percent: null, os_cpu_percent: 42, process_cpu_percent: 7 }))
      .toContain('CPU 42%')
  })

  it('shows CPU N/A when unavailable', () => {
    expect(tooltipFor({ cpu_percent: null, cpu_source: undefined, resource_cpu_percent: null, os_cpu_percent: null, process_cpu_percent: null }))
      .toContain('CPU N/A')
  })
})
