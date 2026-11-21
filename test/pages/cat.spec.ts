import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import CatPage from '@/pages/cat/index.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

describe('cat page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('does not offer the unsupported cluster_manager preset', () => {
    setConnection()
    const wrapper = mount(CatPage)
    const options = wrapper.findAll('select option').map((o) => o.text())
    expect(options).not.toContain('cluster_manager')
    expect(options).toContain('master')
  })

  it('clears a previous result table when a request fails', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: [{ 'health': 'green' }] })
      .mockResolvedValueOnce({ status: 400, body: { error: 'no handler found' } })

    const wrapper = mount(CatPage)
    await flushPromises()

    const execute = wrapper.findAll('button').find((b) => b.text().includes('Execute'))
    await execute!.trigger('click')
    await flushPromises()

    expect(wrapper.find('table.utable').exists()).toBe(true)

    await execute!.trigger('click')
    await flushPromises()

    expect(wrapper.find('table.utable').exists()).toBe(false)
    expect(wrapper.text()).toContain('no handler found')
  })
})
