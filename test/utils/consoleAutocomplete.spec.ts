import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { autocompletion, currentCompletions, startCompletion } from '@codemirror/autocomplete'
import { createConsoleCompletionSource } from '@/utils/consoleAutocomplete'
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

function contextFor(doc: string, pos = doc.length) {
  const state = EditorState.create({ doc })
  return {
    state,
    pos,
    explicit: true,
    matchBefore(re: RegExp) {
      const line = state.doc.lineAt(pos)
      const text = state.sliceDoc(line.from, pos)
      const match = new RegExp(`${re.source}$`).exec(text)
      if (!match) return null
      return { from: pos - match[0].length, to: pos, text: match[0] }
    },
  } as any
}

function meta(overrides: Partial<RestMetadata> = {}): RestMetadata {
  return {
    indices: ['logs-2024', 'metrics'],
    fieldsByIndex: { 'logs-2024': ['message', 'level', 'timestamp'] },
    allFields: ['message', 'level', 'timestamp'],
    backendType: 'elasticsearch',
    host: 'http://es:9200',
    ...overrides,
  }
}

function labels(result: any): string[] {
  return (result?.options || []).map((o: any) => o.label)
}

describe('consoleAutocomplete', () => {
  it('suggests HTTP methods at the start of a line', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GE'))
    expect(labels(result)).toContain('GET')
    expect(labels(result)).toContain('POST')
  })

  it('suggests Elasticsearch lifecycle endpoints', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET _il'))
    expect(labels(result)).toContain('_ilm/policy')
    expect(labels(result)).toContain('_slm/policy')
  })

  it('suggests OpenSearch lifecycle endpoints for OpenSearch backends', () => {
    const source = createConsoleCompletionSource(ref(meta({ backendType: 'opensearch' })))
    const result = source(contextFor('GET _pl'))
    expect(labels(result)).toContain('_plugins/_ism/policies')
    expect(labels(result)).toContain('_plugins/_sm/policies')
  })

  it('suggests mapped fields inside a field-keyed clause', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const doc = 'POST logs-2024/_search\n{"query":{"match":{"'
    const result = source(contextFor(doc))
    expect(labels(result)).toContain('message')
    expect(labels(result)).toContain('level')
  })

  it('suggests query DSL keys inside a query object', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const doc = 'POST logs-2024/_search\n{"query":{"'
    const result = source(contextFor(doc))
    expect(labels(result)).toContain('bool')
    expect(labels(result)).toContain('match')
  })
})

describe('consoleAutocomplete triggers', () => {
  it('suggests query DSL keys inside a multi-line JSON body', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const doc = 'GET logs-2024/_search\n{\n  "query": {\n    "'
    const result = source(contextFor(doc))
    expect(labels(result)).toContain('bool')
    expect(labels(result)).toContain('match')
  })

  it('suggests query params after ? and &', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    expect(labels(source(contextFor('GET logs-2024/_search?pre')))).toContain('pretty')
    expect(labels(source(contextFor('GET logs-2024/_search?pretty&h')))).toContain('h=')
  })

  it('suggests indices and global endpoints right after the method and space', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET '))
    expect(labels(result)).toContain('logs-2024')
    expect(labels(result)).toContain('_cluster/health')
  })

  it('suggests index endpoints after a slash', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET logs-2024/'))
    expect(labels(result)).toContain('_search')
    expect(labels(result)).toContain('_mapping')
  })

  it('supports leading-slash paths', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET /logs-2024/_sea'))
    expect(labels(result)).toContain('_search')
  })

  it('suggests data stream endpoints', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET _data'))
    expect(labels(result)).toContain('_data_stream')
  })
})

async function realCompletions(doc: string, metaValue: RestMetadata): Promise<string[]> {
  const parent = document.createElement('div')
  document.body.appendChild(parent)
  const view = new EditorView({
    parent,
    state: EditorState.create({
      doc,
      extensions: [autocompletion({ activateOnTyping: true, override: [createConsoleCompletionSource(ref(metaValue))] })],
    }),
  })
  view.dispatch({ selection: { anchor: doc.length } })
  startCompletion(view)
  // startCompletion schedules the source with a 50ms debounce.
  await new Promise((resolve) => setTimeout(resolve, 80))
  const labels = currentCompletions(view.state).map((completion) => completion.label)
  view.destroy()
  parent.remove()
  return labels
}

describe('consoleAutocomplete _search ranking (real CodeMirror filtering)', () => {
  const cases = [
    'GET _s',
    'GET _sea',
    'GET /_sea',
    'POST _sea',
    'GET sea',
    'GET my-index/',
    'GET my-index/_s',
    'GET my-index/s',
    'GET _all/_s',
  ]

  for (const backendType of ['elasticsearch', 'opensearch']) {
    for (const doc of cases) {
      it(`offers _search for "${doc}" on ${backendType}`, async () => {
        const labels = await realCompletions(doc, meta({ backendType }))
        expect(labels).toContain('_search')
      })
    }
  }

  it('ranks _search ahead of _msearch for GET _sea', async () => {
    const labels = await realCompletions('GET _sea', meta())
    expect(labels.indexOf('_search')).toBeGreaterThanOrEqual(0)
    expect(labels.indexOf('_search')).toBeLessThan(labels.indexOf('_msearch'))
  })

  it('ranks _search first for GET my-index/s (single-letter non-underscore prefix)', async () => {
    const labels = await realCompletions('GET my-index/s', meta())
    expect(labels[0]).toBe('_search')
  })

  it('offers _search for GET _all/_s', async () => {
    const labels = await realCompletions('GET _all/_s', meta())
    expect(labels).toContain('_search')
  })

  it('offers OpenSearch index point-in-time endpoints', async () => {
    const labels = await realCompletions('GET my-index/_search/point', meta({ backendType: 'opensearch' }))
    expect(labels).toContain('_search/point_in_time')
  })
})

describe('consoleAutocomplete endpoint catalogs', () => {
  it('includes _search in the global endpoint catalog', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    expect(labels(source(contextFor('GET _sea')))).toContain('_search')
  })

  it('routes _all/<endpoint> to index endpoints', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    expect(labels(source(contextFor('GET _all/_s')))).toContain('_search')
  })

  it('deduplicates endpoint options', () => {
    const source = createConsoleCompletionSource(ref(meta()))
    const result = source(contextFor('GET my-index/'))
    const all = labels(result)
    expect(new Set(all).size).toBe(all.length)
  })
})
