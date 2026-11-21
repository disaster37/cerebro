import { describe, it, expect } from 'vitest'
import {
  buildCurlCommand,
  findRequestAtOffset,
  formatRequestBlockBody,
  parseConsoleDocument,
} from '@/utils/consoleParser'

describe('consoleParser', () => {
  it('parses multiple requests and ignores comments', () => {
    const doc = `# comment
GET _cluster/health

// another comment
POST _search
{
  "query": { "match_all": {} }
}
`
    const requests = parseConsoleDocument(doc)
    expect(requests).toHaveLength(2)
    expect(requests[0]!.method).toBe('GET')
    expect(requests[0]!.path).toBe('_cluster/health')
    expect(requests[0]!.rawBody).toBe('')
    expect(requests[1]!.method).toBe('POST')
    expect(requests[1]!.path).toBe('_search')
    expect(requests[1]!.rawBody).toContain('"match_all"')
  })

  it('parses _bulk NDJSON bodies', () => {
    const doc = `POST _bulk
{"index":{"_index":"idx"}}
{"field":"value"}
`
    const requests = parseConsoleDocument(doc)
    expect(requests).toHaveLength(1)
    expect(requests[0]!.isNdjsonEndpoint).toBe(true)
    expect(requests[0]!.rawBody).toContain('{"index"')
    expect(requests[0]!.rawBody).toContain('{"field"')
  })

  it('resolves the request at a cursor offset', () => {
    const doc = `GET _cluster/health\n\nPOST _search\n{"query":{}}\n`
    const requests = parseConsoleDocument(doc)
    const second = requests[1]!
    expect(findRequestAtOffset(requests, second.headerFrom + 2)?.path).toBe('_search')
    expect(findRequestAtOffset(requests, 0)?.path).toBe('_cluster/health')
  })

  it('formats a JSON body block', () => {
    const doc = `POST _search\n{"query":{"match_all":{}}}`
    const request = parseConsoleDocument(doc)[0]!
    const formatted = formatRequestBlockBody(doc, request)
    expect(formatted).not.toBeNull()
    expect(formatted!.insert).toContain('\n  "query"')
  })

  it('builds a cURL command with content type and body', () => {
    const doc = `POST _search\n{"query":{"match_all":{}}}`
    const request = parseConsoleDocument(doc)[0]!
    const curl = buildCurlCommand('http://es:9200', request)
    expect(curl).toContain("curl -X POST 'http://es:9200/_search'")
    expect(curl).toContain("Content-Type: application/json")
    expect(curl).toContain('"match_all"')
  })
})

describe('consoleParser CRLF and comments', () => {
  it('parses CRLF documents and preserves exact offsets', () => {
    const doc = 'GET _cluster/health\r\n\r\nPOST _search\r\n{\r\n  "query": { "match_all": {} }\r\n}\r\n'
    const requests = parseConsoleDocument(doc)
    expect(requests).toHaveLength(2)
    expect(requests[0]!.path).toBe('_cluster/health')
    expect(requests[1]!.path).toBe('_search')
    expect(requests[1]!.rawBody).toContain('"match_all"')
    expect(doc.slice(requests[1]!.headerFrom, requests[1]!.headerTo).replace(/\r$/, '')).toBe('POST _search')
  })

  it('tolerates comment lines before and inside a JSON body', () => {
    const doc = `POST _search
# leading comment
{
  // inside comment
  "query": { "match_all": {} }
}
`
    const requests = parseConsoleDocument(doc)
    expect(requests).toHaveLength(1)
    expect(requests[0]!.rawBody).toContain('"match_all"')
    expect(requests[0]!.rawBody).not.toContain('leading comment')
    expect(requests[0]!.rawBody).not.toContain('inside comment')
    expect(() => JSON.parse(requests[0]!.rawBody)).not.toThrow()
  })
})
