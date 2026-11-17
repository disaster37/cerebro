import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import KibanaConsoleEditor from '@/components/rest/KibanaConsoleEditor.vue'
import DevToolsPage from '@/pages/devtools/index.vue'
import { resetTestStubs } from '../setup'
import type { RestMetadata } from '@/types/cerebro'

// CodeMirror measures text ranges; jsdom does not implement these.
if (!Range.prototype.getClientRects) {
  Range.prototype.getClientRects = () => ({
    length: 0,
    item: () => null,
    [Symbol.iterator]: function* () {},
  }) as any
}
if (!Range.prototype.getBoundingClientRect) {
  Range.prototype.getBoundingClientRect = () => ({
    x: 0, y: 0, top: 0, left: 0, right: 0, bottom: 0, width: 0, height: 0,
    toJSON: () => ({}),
  }) as any
}

const meta: RestMetadata = {
  indices: ['my-index'],
  fieldsByIndex: { 'my-index': ['message'] },
  allFields: ['message'],
  backendType: 'elasticsearch',
  host: 'https://es:9200',
}

function setConnection() {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username: '',
    password: '',
    trustCertificate: false,
  }
  ;(globalThis as any).useState('clusterName').value = 'prod-cluster'
}

describe('KibanaConsoleEditor', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('mounts a real EditorView without throwing and renders hover actions', async () => {
    const wrapper = mount(KibanaConsoleEditor, {
      props: { modelValue: 'GET _cluster/health\n', meta },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="console-editor"] .cm-editor').exists()).toBe(true)
    expect(wrapper.text()).toContain('▶')
    expect(wrapper.text()).toContain('Copy as cURL')
  })

  it('parses and sends a multi-line GET request body', async () => {
    const doc = 'GET my-index/_search\n{\n  "query": { "match_all": {} }\n}\n'
    const wrapper = mount(KibanaConsoleEditor, {
      props: { modelValue: doc, meta },
    })
    await flushPromises()

    const send = wrapper.findAll('.cm-header-actions button').find((b) => b.text().includes('▶ Send'))
    expect(send).toBeTruthy()
    await send!.trigger('mousedown')
    await flushPromises()

    const emitted = wrapper.emitted('send') as any[]
    expect(emitted).toBeTruthy()
    const request = emitted[0]![0]
    expect(request.method).toBe('GET')
    expect(request.path).toBe('my-index/_search')
    expect(request.rawBody).toContain('"match_all"')
  })
})

describe('DevTools page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  function mockSchemaAndHistory() {
    ;(globalThis as any).$fetch.mockImplementation((url: string) => {
      if (url === '/rest') {
        return Promise.resolve({ status: 200, body: { indices: ['my-index'], fieldsByIndex: {}, allFields: [], backendType: 'elasticsearch', host: 'https://es:9200' } })
      }
      if (url === '/rest/history') return Promise.resolve({ status: 200, body: [] })
      if (url === '/rest/request') return Promise.resolve({ status: 200, body: { ok: true } })
      return Promise.resolve({ status: 200, body: {} })
    })
  }

  it('renders the DevTools heading and sends the active request via POST /rest/request', async () => {
    setConnection()
    mockSchemaAndHistory()

    const wrapper = mount(DevToolsPage)
    await flushPromises()

    expect(wrapper.text()).toContain('DevTools')

    const send = wrapper.findAll('.cm-header-actions button').find((b) => b.text().includes('▶ Send'))
    expect(send).toBeTruthy()
    await send!.trigger('mousedown')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/rest/request', expect.objectContaining({
      body: expect.objectContaining({ method: 'GET', path: '_cluster/health' }),
    }))
  })

  it('shows inline feedback after copying a request as cURL', async () => {
    setConnection()
    mockSchemaAndHistory()
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })

    const wrapper = mount(DevToolsPage)
    await flushPromises()

    const copy = wrapper.findAll('.cm-header-actions button').find((b) => b.text() === 'Copy as cURL')
    expect(copy).toBeTruthy()
    await copy!.trigger('mousedown')
    await flushPromises()

    expect(writeText).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="copy-curl-feedback"]').text()).toContain('Copied cURL')
  })
})
