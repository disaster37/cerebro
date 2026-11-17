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
    { name: 'alpha', closed: false, special: false, backing_index: false, status: 'green', unhealthy: false, unreplicated: false, unassigned_shard_count: 0, initializing_shard_count: 0, relocating_shard_count: 0, doc_count: 100, deleted_docs: 2, size_in_bytes: 1024, total_size_in_bytes: 2048, aliases: ['a'], num_shards: 1, num_replicas: 0, shards: {} },
    { name: 'beta', closed: false, special: false, backing_index: false, status: 'yellow', unhealthy: true, unreplicated: false, unassigned_shard_count: 1, initializing_shard_count: 0, relocating_shard_count: 0, doc_count: 50, deleted_docs: 0, size_in_bytes: 512, total_size_in_bytes: 512, aliases: [], num_shards: 1, num_replicas: 0, shards: {} },
    { name: '.hidden', closed: false, special: true, backing_index: false, status: 'green', unhealthy: false, unreplicated: false, unassigned_shard_count: 0, initializing_shard_count: 0, relocating_shard_count: 0, doc_count: 1, deleted_docs: 0, size_in_bytes: 10, total_size_in_bytes: 10, aliases: [], num_shards: 1, num_replicas: 0, shards: {} },
  ],
  data_streams: [],
  nodes: [],
  shard_allocation: true,
}

const dataStreams = [
  {
    name: 'logs-app', status: 'green', generation: 1, template: 'logs-app-ds-template',
    timestamp_field: '@timestamp', write_index: '.ds-logs-app-000001',
    backing_indices: ['.ds-logs-app-000001'], doc_count: 10, size_in_bytes: 100, total_size_in_bytes: 100,
  },
]

function mockLoad(overview: any = overviewBody, streams: any[] = []) {
  ;(globalThis as any).$fetch.mockImplementation((url: string) => {
    if (url === '/overview') return Promise.resolve({ status: 200, body: overview })
    if (url === '/data_streams') return Promise.resolve({ status: 200, body: streams })
    if (url === '/data_streams/create') return Promise.resolve({ status: 200, body: { acknowledged: true } })
    if (url === '/data_streams/rollover') return Promise.resolve({ status: 200, body: { acknowledged: true } })
    if (url === '/data_streams/delete') return Promise.resolve({ status: 200, body: { acknowledged: true } })
    return Promise.resolve({ status: 200, body: { acknowledged: true } })
  })
}

describe('indices page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loads /overview and renders docs and size', async () => {
    setConnection()
    mockLoad()

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
    mockLoad()

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
    mockLoad()

    const wrapper = mount(IndicesPage)
    await flushPromises()

    const header = wrapper.find('[data-testid="sort-doc_count"]')
    expect(header.exists()).toBe(true)
    await header.trigger('click')
    await header.trigger('click')

    const rows = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(rows[0]).toContain('alpha')
  })

  it('runs an index action and reports success feedback', async () => {
    setConnection()
    mockLoad()

    const wrapper = mount(IndicesPage)
    await flushPromises()

    await (wrapper.vm as any).runIndexAction('/overview/flush_indices', 'alpha', 'Flushed index')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/overview/flush_indices', expect.objectContaining({
      body: expect.objectContaining({ indices: 'alpha' }),
    }))
    expect((globalThis as any).useState('alerts').value.some((a: any) => a.message.includes('Flushed index alpha'))).toBe(true)
  })

  it('requires confirmation before deleting an index', async () => {
    setConnection()
    mockLoad()

    const wrapper = mount(IndicesPage)
    await flushPromises()

    ;(wrapper.vm as any).confirmIndexAction('Delete index?', 'Delete', '/overview/delete_indices', 'alpha', 'Deleted index')
    await flushPromises()

    const confirm = wrapper.findAll('button').filter((b) => b.text() === 'Delete').at(-1)
    expect(confirm).toBeTruthy()
    await confirm!.trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/overview/delete_indices', expect.objectContaining({
      body: expect.objectContaining({ indices: 'alpha' }),
    }))
  })

  it('renders the Data Streams tab and drills into backing indices', async () => {
    setConnection()
    const overview = {
      ...overviewBody,
      indices: [
        ...overviewBody.indices,
        { name: '.ds-logs-app-000001', closed: false, special: true, backing_index: true, data_stream: 'logs-app', is_write_index: true, status: 'green', unhealthy: false, unreplicated: false, unassigned_shard_count: 0, initializing_shard_count: 0, relocating_shard_count: 0, doc_count: 10, deleted_docs: 0, size_in_bytes: 100, total_size_in_bytes: 100, aliases: [], num_shards: 1, num_replicas: 0, shards: {} },
      ],
    }
    mockLoad(overview, dataStreams)

    const wrapper = mount(IndicesPage)
    await flushPromises()

    const tab = wrapper.findAll('button').find((b) => b.text().includes('Data Streams (1)'))
    expect(tab).toBeTruthy()
    await tab!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('logs-app')
    expect(wrapper.text()).toContain('View Backing Indices')

    const view = wrapper.findAll('button').find((b) => b.text() === 'View Backing Indices')
    await view!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="datastream-filter-chip"]').text()).toContain('Data stream: logs-app')
    expect(wrapper.text()).toContain('.ds-logs-app-000001')
  })

  it('creates a data stream from the modal', async () => {
    setConnection()
    mockLoad(overviewBody, dataStreams)

    const wrapper = mount(IndicesPage)
    await flushPromises()

    await wrapper.find('[data-testid="open-create-datastream"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-testid="datastream-name"]').setValue('logs-app')
    await wrapper.find('[data-testid="datastream-submit"]').trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/data_streams/create', expect.objectContaining({
      body: expect.objectContaining({ name: 'logs-app', createTemplate: true }),
    }))
  })

  it('renders a disconnected prompt without fetching', async () => {
    const wrapper = mount(IndicesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="indices-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('No active cluster connection')
  })
})
