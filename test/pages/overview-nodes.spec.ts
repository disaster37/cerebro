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
