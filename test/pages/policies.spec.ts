import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import PoliciesPage from '@/pages/policies/index.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

const ilmPayload = {
  backendType: 'elasticsearch',
  featureName: 'ILM',
  operationMode: 'RUNNING',
  policies: [
    { id: 'p1', version: 1, modified_date: '2024-01-01', phases: ['hot', 'delete'], managed_count: 1, failed_count: 1, raw_policy: { policy: {} } },
  ],
  managedIndices: [
    { index: 'idx1', policy_id: 'p1', phase: 'hot', action: 'rollover', step: 'ERROR', failed: true, failed_step: 'check', retry_count: 0, reason: 'boom' },
  ],
}

const slmPayload = [
  {
    id: 'nightly', name: 'nightly', repository: 'repo', schedule: '0 30 1 * * ?', indices: ['*'],
    enabled: true, current_state: 'RUNNING', last_failure_message: 's3 timeout',
    snapshots_taken: 5, snapshots_failed: 1, raw_policy: {},
  },
]

const ismPayload = {
  backendType: 'opensearch',
  featureName: 'ISM',
  operationMode: 'RUNNING',
  policies: [
    { id: 'ism1', version: 1, modified_date: '', phases: ['hot', 'delete'], managed_count: 1, failed_count: 1, raw_policy: {} },
  ],
  managedIndices: [
    { index: 'idx1', policy_id: 'ism1', phase: 'hot', action: 'rollover', step: 'check', failed: true, retry_count: 2, reason: 'disk full', available_states: ['hot', 'delete'] },
  ],
}

function mockLoad(indexBody: any, snapshotBody: any) {
  ;(globalThis as any).$fetch.mockImplementation((url: string) => {
    if (url === '/policies/index_lifecycle') return Promise.resolve({ status: 200, body: indexBody })
    if (url === '/policies/snapshot_lifecycle') return Promise.resolve({ status: 200, body: snapshotBody })
    return Promise.resolve({ status: 200, body: {} })
  })
}

describe('policies page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('renders ILM failures and SLM policies', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    expect(wrapper.text()).toContain('Lifecycle Policies (ILM & SLM)')
    expect(wrapper.text()).toContain('idx1')
    expect(wrapper.text()).toContain('FAILED')
    expect(wrapper.text()).toContain('boom')

    await wrapper.findAll('button').find((b) => b.text().includes('Snapshot Policies'))!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('nightly')
    expect(wrapper.text()).toContain('s3 timeout')
  })

  it('retries an ILM step', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    const retry = wrapper.findAll('button').find((b) => b.text() === 'Retry Step')
    expect(retry).toBeTruthy()
    await retry!.trigger('click')
    await flushPromises()

    const confirm = wrapper.findAll('button').filter((b) => b.text() === 'Retry').at(-1)
    expect(confirm).toBeTruthy()
    await confirm!.trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/policies/retry_indices', expect.objectContaining({
      body: { host: 'https://es:9200', indices: ['idx1'] },
    }))
  })

  it('executes an SLM policy', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text().includes('Snapshot Policies'))!.trigger('click')
    await flushPromises()

    const run = wrapper.findAll('button').find((b) => b.text() === 'Run Now / Retry')
    expect(run).toBeTruthy()
    await run!.trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/policies/snapshot_action', expect.objectContaining({
      body: { host: 'https://es:9200', policyId: 'nightly', action: 'execute' },
    }))
  })

  it('retries an OpenSearch ISM policy from a specific state', async () => {
    setConnection()
    mockLoad(ismPayload, [])

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    expect(wrapper.text()).toContain('State & Snapshot Management (ISM & SM)')
    const retry = wrapper.findAll('button').find((b) => b.text() === 'Retry Policy')
    expect(retry).toBeTruthy()
    await retry!.trigger('click')
    await flushPromises()

    const fromState = wrapper.findAll('input[type="checkbox"]').find((el) => el.element.parentElement?.textContent?.includes('Restart from specific state'))
    expect(fromState).toBeTruthy()
    await fromState!.setValue(true)

    const retryConfirm = wrapper.findAll('button').filter((b) => b.text() === 'Retry').at(-1)
    expect(retryConfirm).toBeTruthy()
    await retryConfirm!.trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/policies/retry_indices', expect.objectContaining({
      body: { host: 'https://es:9200', indices: ['idx1'], state: 'hot' },
    }))
  })

  it('renders a disconnected prompt without fetching', async () => {
    const wrapper = mount(PoliciesPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="policies-error"]').exists()).toBe(true)
  })
})

describe('policies actions', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('renders labeled Details and View JSON buttons instead of bare icon-only braces', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    expect(wrapper.findAll('button').some((b) => b.text() === 'Details')).toBe(true)
    expect(wrapper.findAll('button').some((b) => b.text() === 'View JSON')).toBe(true)
    expect(wrapper.text()).not.toContain('{}')
  })

  it('opens the policy JSON inspector from the View JSON button', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    const view = wrapper.findAll('button').find((b) => b.text() === 'View JSON')
    await view!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Policy: p1')
  })

  it('renders snapshot policy action icons and View JSON', async () => {
    setConnection()
    mockLoad(ilmPayload, slmPayload)

    const wrapper = mount(PoliciesPage)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text().includes('Snapshot Policies'))!.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('button').some((b) => b.text() === 'Run Now / Retry')).toBe(true)
    expect(wrapper.findAll('button').some((b) => b.text() === 'View JSON')).toBe(true)
  })
})
