import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import IndicesPage from '@/pages/indices/index.vue'
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

const overviewBody = {
  cluster_name: 'prod',
  status: 'green',
  number_of_nodes: 2,
  active_shards: 4,
  unassigned_shards: 1,
  docs_count: 1234,
  size_in_bytes: 2048,
  total_indices: 3,
  closed_indices: 1,
  special_indices: 1,
  indices: [
    { name: 'alpha', closed: false, special: false, unhealthy: false, doc_count: 100, deleted_docs: 2, size_in_bytes: 1024, total_size_in_bytes: 2048, aliases: ['a'], num_shards: 1, num_replicas: 0, shards: {} },
    { name: 'beta', closed: false, special: false, unhealthy: true, doc_count: 50, deleted_docs: 0, size_in_bytes: 512, total_size_in_bytes: 512, aliases: [], num_shards: 1, num_replicas: 0, shards: {} },
    { name: '.hidden', closed: false, special: true, unhealthy: false, doc_count: 1, deleted_docs: 0, size_in_bytes: 10, total_size_in_bytes: 10, aliases: [], num_shards: 1, num_replicas: 0, shards: {} },
  ],
  nodes: [],
  shard_allocation: true,
}

describe('indices page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loads /overview and renders docs and size', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: overviewBody })

    const wrapper = mount(IndicesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/overview', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'admin', password: 'secret' },
    }))
    expect(wrapper.text()).toContain('1,234')
    expect(wrapper.text()).toContain('2 KB')
    expect(wrapper.text()).toContain('alpha')
  })

  it('hides special indices by default and shows them when toggled', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: overviewBody })

    const wrapper = mount(IndicesPage)
    await flushPromises()
    expect(wrapper.text()).not.toContain('.hidden')

    const special = wrapper.findAll('input[type="checkbox"]').find((el) => el.element.parentElement?.textContent?.includes('.special'))
    expect(special).toBeTruthy()
    await special!.setValue(true)
    expect(wrapper.text()).toContain('.hidden')
  })

  it('sorts by docs descending when the header is clicked twice', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: overviewBody })

    const wrapper = mount(IndicesPage)
    await flushPromises()

    const header = wrapper.find('[data-testid="sort-doc_count"]')
    expect(header.exists()).toBe(true)
    await header.trigger('click')
    await header.trigger('click')

    const rows = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(rows[0]).toContain('alpha')
  })

  it('renders a disconnected prompt without fetching', async () => {
    const wrapper = mount(IndicesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="indices-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No active cluster connection')
  })
})
