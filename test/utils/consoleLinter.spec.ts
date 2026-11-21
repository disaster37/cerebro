import { describe, it, expect } from 'vitest'
import { validateConsoleDocument } from '@/utils/consoleLinter'

describe('consoleLinter', () => {
  it('accepts a valid document', () => {
    const doc = `GET _cluster/health\n\nPOST _search\n{"query":{"match_all":{}}}\n`
    expect(validateConsoleDocument(doc)).toHaveLength(0)
  })

  it('flags an unrecognized HTTP method', () => {
    const diagnostics = validateConsoleDocument('GT _search\n')
    expect(diagnostics.some((d) => d.severity === 'error' && d.message.includes('Unrecognized HTTP method'))).toBe(true)
  })

  it('flags a missing path', () => {
    const diagnostics = validateConsoleDocument('GET   \n')
    expect(diagnostics.some((d) => d.message.includes('Missing request path'))).toBe(true)
  })

  it('flags orphan text', () => {
    const diagnostics = validateConsoleDocument('this is not a request\n')
    expect(diagnostics.some((d) => d.message.includes('Orphan text'))).toBe(true)
  })

  it('flags broken JSON with an offset', () => {
    const doc = 'POST _search\n{"query": {"match_all": {}}\n'
    const diagnostics = validateConsoleDocument(doc)
    const error = diagnostics.find((d) => d.severity === 'error' && d.message.toLowerCase().includes('json'))
    expect(error).toBeTruthy()
    expect(error!.from).toBeGreaterThanOrEqual(doc.indexOf('{'))
  })

  it('flags a broken NDJSON line', () => {
    const doc = 'POST _bulk\n{"index":{"_index":"idx"}}\n{not json}\n'
    const diagnostics = validateConsoleDocument(doc)
    expect(diagnostics.some((d) => d.message.includes('Invalid NDJSON line'))).toBe(true)
  })

  it('warns when HEAD has a body', () => {
    const doc = 'HEAD _bulk\n{"index":{}}\n'
    const diagnostics = validateConsoleDocument(doc)
    expect(diagnostics.some((d) => d.severity === 'warning')).toBe(true)
  })
})
