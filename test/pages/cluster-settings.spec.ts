import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ClusterSettingsPage from '@/pages/cluster-settings.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

const settingsBody = {
  defaults: {
    'cluster.max_shards_per_node': '1000',
    'cluster.routing.allocation.disk.threshold_enabled': 'true',
  },
  persistent: {
    'cluster.max_shards_per_node': '2000',
  },
  transient: {
    'cluster.max_shards_per_node': '3000',
  },
}

describe('cluster settings page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loads settings and shows the effective transient value', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: settingsBody })

    const wrapper = mount(ClusterSettingsPage)
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/cluster_settings', expect.objectContaining({
      body: { host: 'https://es:9200' },
    }))
    expect(wrapper.text()).toContain('Shard Allocation & Disk Watermarks')
    expect(wrapper.text()).toContain('transient')
    const input = wrapper.findAll('input').find((el) => (el.element as HTMLInputElement).value === '3000')
    expect(input).toBeTruthy()
  })

  it('rejects an unknown extra setting key', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: settingsBody })

    const wrapper = mount(ClusterSettingsPage)
    await flushPromises()

    await wrapper.find('[data-testid="extra-setting-key"]').setValue('bogus.setting.key')
    expect(wrapper.find('[data-testid="extra-setting-validation"]').text()).toContain('does not exist')
    expect(wrapper.find('[data-testid="stage-extra-setting"]').attributes('disabled')).toBeDefined()
  })

  it('stages a known extra setting and saves it under the selected scope', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: settingsBody })
      .mockResolvedValueOnce({ status: 200, body: { acknowledged: true } })
      .mockResolvedValueOnce({ status: 200, body: settingsBody })

    const wrapper = mount(ClusterSettingsPage)
    await flushPromises()

    await wrapper.find('[data-testid="extra-setting-key"]').setValue('cluster.max_shards_per_node')
    await wrapper.find('[data-testid="extra-setting-value"]').setValue('4000')
    await wrapper.find('[data-testid="stage-extra-setting"]').trigger('click')

    expect(wrapper.find('[data-testid="pending-changes"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('4000')

    await wrapper.find('[data-testid="save-cluster-settings"]').trigger('click')
    await flushPromises()

    const saveCall = (globalThis as any).$fetch.mock.calls.find((call: any[]) => call[0] === '/cluster_settings/save')
    expect(saveCall).toBeTruthy()
    expect(saveCall[1].body.settings.persistent['cluster.max_shards_per_node']).toBe(4000)
  })

  it('surfaces a save error without wiping staged changes', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: settingsBody })
      .mockResolvedValueOnce({ status: 400, body: { error: 'not dynamically updateable' } })

    const wrapper = mount(ClusterSettingsPage)
    await flushPromises()

    await wrapper.find('[data-testid="extra-setting-key"]').setValue('cluster.max_shards_per_node')
    await wrapper.find('[data-testid="extra-setting-value"]').setValue('4000')
    await wrapper.find('[data-testid="stage-extra-setting"]').trigger('click')
    await wrapper.find('[data-testid="save-cluster-settings"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="cluster-settings-save-error"]').text()).toContain('not dynamically updateable')
    expect(wrapper.find('[data-testid="pending-changes"]').exists()).toBe(true)
  })

  it('renders a disconnected prompt without fetching', async () => {
    const wrapper = mount(ClusterSettingsPage)
    await flushPromises()

    expect((globalThis as any).$fetch).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="cluster-settings-error"]').exists()).toBe(true)
  })
})
