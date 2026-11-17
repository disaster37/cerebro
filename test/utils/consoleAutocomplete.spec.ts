import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { EditorState } from '@codemirror/state'
import { createConsoleCompletionSource } from '@/utils/consoleAutocomplete'
import type { RestMetadata } from '@/types/cerebro'

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
