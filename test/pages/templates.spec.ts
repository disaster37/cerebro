import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import TemplatesPage from '@/pages/templates.vue'
import { resetTestStubs } from '../setup'

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

const templates = [
  { name: 'logs-ds', type: 'composable', patterns: ['logs-*'], order: 200, data_stream: true, template: { index_patterns: ['logs-*'], data_stream: {} } },
  { name: 'legacy', type: 'legacy', patterns: ['metrics-*'], order: 1, template: { index_patterns: ['metrics-*'] } },
]

describe('templates page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('renders templates with a data stream badge and labeled View JSON button', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: templates })

    const wrapper = mount(TemplatesPage)
    await flushPromises()

    expect(wrapper.text()).toContain('logs-ds')
    expect(wrapper.text()).toContain('data stream')
    expect(wrapper.findAll('button').some((b) => b.text() === 'View JSON')).toBe(true)
  })

  it('switches the default JSON skeleton between composable and legacy', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: templates })

    const wrapper = mount(TemplatesPage)
    await flushPromises()

    await wrapper.findAll('button').find((b) => b.text() === 'Create Template')!.trigger('click')
    await flushPromises()

    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toContain('"template"')

    const typeSelect = wrapper.find('.umodal select')
    expect(typeSelect.exists()).toBe(true)
    await typeSelect.setValue('legacy')
    await flushPromises()

    expect(wrapper.find('textarea').element.value).not.toContain('"template"')
    expect(wrapper.find('textarea').element.value).toContain('"settings"')
  })

  it('suppresses the global alert on load and renders an inline error card', async () => {
    setConnection()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 500, body: { error: 'boom' } })

    const wrapper = mount(TemplatesPage)
    await flushPromises()

    expect(wrapper.find('[data-testid="templates-error"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('boom')
    expect((globalThis as any).useState('alerts').value).toHaveLength(0)
  })
})
