import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RepositoryPage from '@/pages/repository.vue'
import SnapshotPage from '@/pages/snapshot/index.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

const repositories = [
  { name: 'backups', type: 'fs', settings: { location: '/tmp/backups' } },
]

const snapshots = [
  {
    snapshot: 'snap-1',
    uuid: 'u1',
    version: '9.4.3',
    indices: ['idx1', '.system'],
    data_streams: [],
    include_global_state: true,
    state: 'SUCCESS',
    start_time_in_millis: 1700000000000,
    end_time_in_millis: 1700000001000,
    duration_in_millis: 1000,
    failures: [],
    shards: { total: 2, failed: 0, successful: 2 },
  },
]

describe('repositories page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loads repositories and verifies one on click', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: repositories })
      .mockResolvedValueOnce({
        status: 200,
        body: {
          name: 'backups', verified: true, status: 'CONNECTED', node_count: 1, nodes: [{ id: 'n1', name: 'node-1' }],
          snapshot_count: 2, active_count: 0, failed_count: 0, size_computed: true, size_in_bytes: 1024, logical_size_in_bytes: 2048,
        },
      })

    const wrapper = mount(RepositoryPage)
    await flushPromises()

    expect(wrapper.text()).toContain('backups')
    const verifyButton = wrapper.findAll('button').find((b) => b.text() === 'Verify')
    expect(verifyButton).toBeTruthy()
    await verifyButton!.trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/repositories/verify', expect.objectContaining({
      body: { host: 'https://es:9200', name: 'backups', forceFullSize: false },
    }))
    expect(wrapper.text()).toContain('Connected (1 nodes)')
    expect(wrapper.text()).toContain('1 KB')
  })

  it('surfaces a verification failure reason', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: repositories })
      .mockResolvedValueOnce({ status: 200, body: { name: 'backups', verified: false, status: 'ERROR', node_count: 0, nodes: [], snapshot_count: 0, active_count: 0, failed_count: 0, size_computed: false, size_in_bytes: null, logical_size_in_bytes: null, error: 'bucket not found' } })

    const wrapper = mount(RepositoryPage)
    await flushPromises()
    await wrapper.findAll('button').find((b) => b.text() === 'Verify')!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Verification failed')
    expect(wrapper.text()).toContain('bucket not found')
  })
})

describe('snapshots page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loads repositories and snapshots', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: { repositories: ['backups'], indices: [{ name: 'idx1', special: false }] } })
      .mockResolvedValueOnce({ status: 200, body: snapshots })

    const wrapper = mount(SnapshotPage)
    await flushPromises()

    expect(wrapper.text()).toContain('snap-1')
    expect(wrapper.text()).toContain('SUCCESS')
  })

  it('requires an index selection in specific-indices restore mode', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: { repositories: ['backups'], indices: [{ name: 'idx1', special: false }] } })
      .mockResolvedValueOnce({ status: 200, body: snapshots })

    const wrapper = mount(SnapshotPage)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Restore')!.trigger('click')
    await flushPromises()

    const specific = wrapper.findAll('input[type="checkbox"]').find((el) => el.element.parentElement?.textContent?.includes('Select specific indices'))
    expect(specific).toBeTruthy()
    await specific!.setValue(true)

    expect(wrapper.find('[data-testid="restore-validation"]').text()).toContain('Select at least one index')
    expect(wrapper.find('[data-testid="restore-submit"]').attributes('disabled')).toBeDefined()
  })

  it('surfaces restore errors inside the modal', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: { repositories: ['backups'], indices: [{ name: 'idx1', special: false }] } })
      .mockResolvedValueOnce({ status: 200, body: snapshots })
      .mockResolvedValueOnce({ status: 400, body: { error: { reason: 'cannot restore open index' } } })

    const wrapper = mount(SnapshotPage)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Restore')!.trigger('click')
    await flushPromises()

    await wrapper.find('[data-testid="restore-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="restore-error"]').text()).toContain('cannot restore open index')
  })
})
