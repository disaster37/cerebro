import {
  snippetCompletion,
  type Completion,
  type CompletionContext,
  type CompletionSource,
} from '@codemirror/autocomplete'
import type { Ref } from 'vue'
import type { RestMetadata } from '../types/cerebro'
import { findRequestAtOffset, parseConsoleDocument, type ParsedConsoleRequest } from './consoleParser'

const METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'HEAD', 'PATCH']

const GLOBAL_ENDPOINTS = [
  '_all', '_cluster/health', '_cluster/state', '_cluster/settings',
  '_cluster/allocation/explain', '_cat/indices', '_cat/nodes', '_cat/aliases',
  '_cat/shards', '_cat/templates', '_nodes', '_nodes/stats', '_snapshot',
  '_tasks', '_reindex', '_bulk', '_msearch', '_index_template',
  '_component_template', '_ingest/pipeline', '_data_stream', '_data_stream/*',
]

const INDEX_ENDPOINTS = [
  '_search', '_count', '_doc', '_mapping', '_settings', '_stats', '_aliases',
  '_analyze', '_refresh', '_flush', '_forcemerge', '_cache/clear', '_open',
  '_close', '_update_by_query', '_delete_by_query', '_recovery', '_shards',
  '_rollover',
]

const ES_ENDPOINTS = [
  '_cat/master', '_ilm/policy', '_ilm/explain', '_ilm/status', '_slm/policy',
  '_slm/status', '_slm/stats', '_query',
]

const OS_ENDPOINTS = [
  '_cat/cluster_manager', '_cat/segment_replication', '_plugins/_ism/policies',
  '_plugins/_ism/explain', '_plugins/_ism/retry', '_plugins/_sm/policies',
  '_plugins/_sql', '_plugins/_ppl',
]

const GLOBAL_PREFIXES = ['_cat/', '_cluster/', '_nodes/', '_ilm/', '_slm/', '_data_stream', '_plugins/']

const QUERY_PARAMS = [
  'pretty', 'human=true', 'v=true', 'format=json', 'h=', 's=',
  'flat_settings=true', 'include_defaults=true', 'ignore_unavailable=true',
  'wait_for_completion=true', 'only_errors=true', 'only_managed=true', 'bytes=b',
]

const ROOT_DSL = ['query', 'aggs', 'size', 'from', 'sort', '_source', 'track_total_hits', 'highlight', 'search_after']
const QUERY_DSL = [
  'bool', 'match', 'match_phrase', 'multi_match', 'term', 'terms', 'range',
  'exists', 'prefix', 'wildcard', 'regexp', 'nested', 'match_all', 'ids',
  'query_string', 'simple_query_string', 'knn',
]
const BOOL_DSL = ['must', 'filter', 'should', 'must_not', 'minimum_should_match']
const RANGE_DSL = ['gte', 'gt', 'lte', 'lt']
const AGG_DSL = [
  'terms', 'date_histogram', 'histogram', 'range', 'filters', 'avg', 'sum',
  'min', 'max', 'cardinality', 'percentiles', 'top_hits', 'composite',
]
const MAPPING_TYPES = ['keyword', 'text', 'long', 'integer', 'double', 'boolean', 'date', 'ip', 'object', 'nested', 'geo_point']
const FIELD_CLAUSES = ['match', 'match_phrase', 'term', 'terms', 'range', 'exists', 'prefix', 'wildcard', 'regexp', 'sort', 'field', '_source']

const EMPTY_META: RestMetadata = {
  indices: [],
  fieldsByIndex: {},
  allFields: [],
  backendType: '',
  host: '',
}

function toOptions(values: string[], type = 'keyword'): Completion[] {
  return values.map((value) => ({ label: value, type }))
}

function methodOptions(): Completion[] {
  return [
    ...toOptions(METHODS, 'keyword'),
    snippetCompletion('GET ${1:_search}\n{\n  "query": {\n    "match_all": {}\n  }\n}', {
      label: 'GET _search (template)',
      type: 'keyword',
    }),
    snippetCompletion('POST ${1:_bulk}\n{"index":{"_index":"${2:index}"}}\n{"field":"value"}\n', {
      label: 'POST _bulk (template)',
      type: 'keyword',
    }),
    snippetCompletion('PUT ${1:_data_stream/${2:logs-app}}', {
      label: 'PUT _data_stream (template)',
      type: 'keyword',
    }),
  ]
}

function flavorEndpoints(meta: RestMetadata): string[] {
  return meta.backendType === 'opensearch' ? OS_ENDPOINTS : ES_ENDPOINTS
}

function pathCompletion(meta: RestMetadata, context: CompletionContext, path: string): Completion {
  const pathStart = context.pos - path.length
  const leadingSlash = path.startsWith('/')
  const raw = leadingSlash ? path.slice(1) : path
  const rawStart = pathStart + (leadingSlash ? 1 : 0)
  const segments = raw.split('/')
  const first = segments[0] || ''
  const flavor = flavorEndpoints(meta)

  // Global endpoint prefix (e.g. _cat/, _cluster/, _data_stream, _plugins/).
  if (first.startsWith('_')) {
    const all = [...GLOBAL_ENDPOINTS, ...flavor]
    const knownPrefix = GLOBAL_PREFIXES.find((prefix) => raw.startsWith(prefix))
    const matches = knownPrefix
      ? all.filter((endpoint) => endpoint.startsWith(knownPrefix))
      : all
    return { from: rawStart, options: toOptions(matches, 'property') }
  }

  // <index-or-data-stream>/<endpoint>
  if (segments.length > 1) {
    const endpointStart = rawStart + first.length + 1
    return { from: endpointStart, options: toOptions([...INDEX_ENDPOINTS, ...flavor], 'property') }
  }

  // First path segment: indices, aliases, data streams + global endpoints.
  return {
    from: rawStart,
    options: toOptions([...(meta.indices || []), ...GLOBAL_ENDPOINTS, ...flavor], 'property'),
  }
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function fieldsFor(meta: RestMetadata, path: string): string[] {
  const first = path.replace(/^\//, '').split('/')[0] || ''
  const patterns = first.split(',').map((p) => p.trim()).filter(Boolean)
  const out = new Set<string>()
  for (const pattern of patterns) {
    const exact = meta.fieldsByIndex?.[pattern]
    if (exact?.length) {
      exact.forEach((f) => out.add(f))
      continue
    }
    if (pattern.includes('*')) {
      const re = new RegExp(`^${pattern.split('*').map(escapeRegExp).join('.*')}$`)
      for (const [index, fields] of Object.entries(meta.fieldsByIndex || {})) {
        if (re.test(index)) fields.forEach((f) => out.add(f))
      }
    }
  }
  if (out.size) return [...out]
  return meta.allFields || []
}

function insideString(doc: string, start: number, pos: number): boolean {
  let inString = false
  let escaped = false
  for (let i = start; i < pos; i++) {
    const ch = doc[i]
    if (inString) {
      if (escaped) escaped = false
      else if (ch === '\\') escaped = true
      else if (ch === '"') inString = false
    } else if (ch === '"') {
      inString = true
    }
  }
  return inString
}

// lastJsonKey returns the nearest enclosing object key at the cursor, ignoring
// strings. It lets the completion source know whether the cursor is inside
// `query`, `bool`, `match`, `aggs`, etc.
function lastJsonKey(doc: string, start: number, pos: number): string {
  const stack: Array<{ key: string | null }> = []
  let inString = false
  let escaped = false
  let stringStart = -1
  let pendingKey: string | null = null

  for (let i = start; i < pos; i++) {
    const ch = doc[i]
    if (inString) {
      if (escaped) {
        escaped = false
      } else if (ch === '\\') {
        escaped = true
      } else if (ch === '"') {
        inString = false
        let j = i + 1
        while (j < pos && /\s/.test(doc[j]!)) j++
        if (doc[j] === ':') pendingKey = doc.slice(stringStart + 1, i)
      }
      continue
    }
    if (ch === '"') {
      inString = true
      stringStart = i
    } else if (ch === '{' || ch === '[') {
      stack.push({ key: null })
      pendingKey = null
    } else if (ch === '}' || ch === ']') {
      stack.pop()
      pendingKey = null
    } else if (ch === ',') {
      pendingKey = null
    } else if (ch === ':') {
      const top = stack[stack.length - 1]
      if (top && pendingKey !== null) top.key = pendingKey
      pendingKey = null
    }
  }

  for (let i = stack.length - 1; i >= 0; i--) {
    if (stack[i]!.key) return stack[i]!.key!
  }
  return ''
}

function keyOptions(values: string[], inString: boolean, type = 'property'): Completion[] {
  if (inString) return toOptions(values, type)
  return values.map((value) => snippetCompletion(`"${value}": \${1}`, {
    label: `"${value}"`,
    type,
  }))
}

function bodyCompletion(meta: RestMetadata, context: CompletionContext, request: ParsedConsoleRequest): Completion | null {
  const doc = context.state.doc.toString()
  const bodyStart = request.bodyFrom ?? request.headerTo
  const line = context.state.doc.lineAt(context.pos)
  const lineBefore = line.text.slice(0, context.pos - line.from)
  const inString = insideString(doc, bodyStart, context.pos)
  const trigger = inString || /[{\[,]\s*$/.test(lineBefore) || /\s$/.test(lineBefore) || context.explicit
  if (!trigger) return null

  const word = context.matchBefore(/[\w.-]*/) || { from: context.pos, to: context.pos, text: '' }
  const lastKey = lastJsonKey(doc, bodyStart, context.pos)

  if (lastKey === 'bool' || ['must', 'filter', 'should', 'must_not'].includes(lastKey)) {
    return { from: word.from, options: keyOptions(BOOL_DSL, inString) }
  }
  if (lastKey === 'range') {
    return { from: word.from, options: keyOptions(RANGE_DSL, inString) }
  }
  if (lastKey === 'aggs' || lastKey === 'aggregations') {
    return { from: word.from, options: keyOptions(AGG_DSL, inString) }
  }
  if (lastKey === 'query') {
    return { from: word.from, options: keyOptions(QUERY_DSL, inString) }
  }
  if (FIELD_CLAUSES.includes(lastKey)) {
    return { from: word.from, options: keyOptions(fieldsFor(meta, request.path), inString, 'variable') }
  }
  if (lastKey === 'type') {
    return { from: word.from, options: keyOptions(MAPPING_TYPES, inString, 'type') }
  }
  return { from: word.from, options: keyOptions(ROOT_DSL, inString) }
}

export function createConsoleCompletionSource(metaRef: Ref<RestMetadata>): CompletionSource {
  return (context: CompletionContext) => {
    const doc = context.state.doc.toString()
    const requests = parseConsoleDocument(doc)
    const request = findRequestAtOffset(requests, context.pos)
    const line = context.state.doc.lineAt(context.pos)
    const before = line.text.slice(0, context.pos - line.from)
    const meta = metaRef.value || EMPTY_META

    // JSON body context (multi-line aware: uses the parsed request, not just
    // the current line).
    if (request && context.pos > request.headerTo) {
      return bodyCompletion(meta, context, request)
    }

    // Method token.
    if (/^\s*[A-Za-z]*$/.test(before)) {
      const word = context.matchBefore(/[A-Za-z]*/) || { from: context.pos, to: context.pos, text: '' }
      return { from: word.from, options: methodOptions() }
    }

    // Query params must be checked before the path token (a path containing
    // `?` also matches the path regex).
    if (/[?&]([^&#\s]*)$/.test(before)) {
      const word = context.matchBefore(/[\w=.-]*/) || { from: context.pos, to: context.pos, text: '' }
      return { from: word.from, options: toOptions(QUERY_PARAMS, 'property') }
    }

    // Path token.
    const pathMatch = /^\s*[A-Za-z]+\s+(\S*)$/.exec(before)
    if (pathMatch) {
      return pathCompletion(meta, context, pathMatch[1] || '')
    }

    return null
  }
}
