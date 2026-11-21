import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import IndexSettingsPage from '@/pages/index-settings.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

describe('index settings page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('keeps the success alert visible after the post-save reload', async () => {
    setConnection()
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: { 'my-index': { settings: { 'index.number_of_replicas': '1' } } } })
      .mockResolvedValueOnce({ status: 200, body: { acknowledged: true } })
      .mockResolvedValueOnce({ status: 200, body: { 'my-index': { settings: { 'index.number_of_replicas': '2' } } } })

    const wrapper = mount(IndexSettingsPage)
    await flushPromises()

    await wrapper.find('[data-testid="index-settings-index"]').setValue('my-index')
    const loadButton = wrapper.findAll('button').find((b) => b.text().includes('Load Settings'))
    await loadButton!.trigger('click')
    await flushPromises()

    const settingInput = wrapper.findAll('input').find(
      (el) => (el.element as HTMLInputElement).value === '1',
    )
    expect(settingInput).toBeTruthy()
    await settingInput!.setValue('2')

    await wrapper.find('[data-testid="index-settings-save"]').trigger('click')
    await flushPromises()

    const updateCall = (globalThis as any).$fetch.mock.calls.find((call: any[]) => call[0] === '/index_settings/update')
    expect(updateCall).toBeTruthy()
    expect(updateCall[1].body).toEqual(expect.objectContaining({
      host: 'https://es:9200',
      index: 'my-index',
      settings: { 'index.number_of_replicas': '2' },
    }))

    expect(wrapper.text()).toContain('Index settings saved.')
  })
})
