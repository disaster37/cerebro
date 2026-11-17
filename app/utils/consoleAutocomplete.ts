import {
  snippetCompletion,
  type Completion,
  type CompletionContext,
  type CompletionSource,
} from '@codemirror/autocomplete'
import type { Ref } from 'vue'
import type { RestMetadata } from '../types/cerebro'
import { findRequestAtOffset, parseConsoleDocument } from './consoleParser'

const METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'HEAD', 'PATCH']

const GLOBAL_ENDPOINTS = [
  '_all', '_cluster/health', '_cluster/state', '_cluster/settings', '_cat/indices',
  '_cat/nodes', '_nodes/stats', '_snapshot', '_tasks', '_reindex', '_bulk',
  '_msearch', '_index_template', '_component_template', '_ingest/pipeline',
]

const INDEX_ENDPOINTS = [
  '_search', '_count', '_doc', '_mapping', '_settings', '_stats', '_aliases',
  '_analyze', '_refresh', '_flush', '_forcemerge', '_cache/clear', '_open',
  '_close', '_update_by_query', '_delete_by_query', '_recovery', '_shards',
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
  ]
}

function pathOptions(meta: RestMetadata, before: string): Completion[] {
  const parts = before.trim().split(/\s+/)
  const path = parts[1] || ''
  const segments = path.split('/')
  const flavor = meta.backendType === 'opensearch' ? OS_ENDPOINTS : ES_ENDPOINTS

  if (segments.length > 1 && segments[0]) {
    return toOptions([...INDEX_ENDPOINTS, ...flavor], 'property')
  }
  return toOptions([...(meta.indices || []), ...GLOBAL_ENDPOINTS, ...flavor], 'property')
}

function bodyOptions(meta: RestMetadata, context: CompletionContext): Completion[] {
  const doc = context.state.doc.toString()
  const requests = parseConsoleDocument(doc)
  const request = findRequestAtOffset(requests, context.pos)
  const before = context.state.sliceDoc(Math.max(0, context.pos - 400), context.pos)

  const keyMatches = [...before.matchAll(/"([^"]+)"\s*:/g)]
  const lastKey = keyMatches.length ? keyMatches[keyMatches.length - 1]![1]! : ''

  if (lastKey === 'bool') return toOptions(BOOL_DSL, 'property')
  if (lastKey === 'range') return toOptions(RANGE_DSL, 'property')
  if (lastKey === 'aggs' || lastKey === 'aggregations') return toOptions(AGG_DSL, 'property')
  if (lastKey === 'query') return toOptions(QUERY_DSL, 'property')
  if (['match', 'term', 'terms', 'sort', '_source', 'exists', 'prefix', 'wildcard', 'regexp'].includes(lastKey)) {
    return toOptions(fieldsFor(meta, request?.path || ''), 'variable')
  }
  if (lastKey === 'type') return toOptions(MAPPING_TYPES, 'type')
  return toOptions(ROOT_DSL, 'property')
}

function fieldsFor(meta: RestMetadata, path: string): string[] {
  const index = path.split('/')[0] || ''
  const fields = meta.fieldsByIndex?.[index]
  if (fields?.length) return fields
  return meta.allFields || []
}

export function createConsoleCompletionSource(metaRef: Ref<RestMetadata>): CompletionSource {
  return (context: CompletionContext) => {
    const word = context.matchBefore(/[\w.*-]*/)
    if (!word || (word.from === word.to && !context.explicit)) return null

    const line = context.state.doc.lineAt(context.pos)
    const before = line.text.slice(0, context.pos - line.from)
    const meta = metaRef.value || {
      indices: [],
      fieldsByIndex: {},
      allFields: [],
      backendType: '',
      host: '',
    }

    if (/^\s*[A-Za-z]*$/.test(before)) {
      return { from: word.from, options: methodOptions() }
    }
    if (/^\s*[A-Za-z]+\s+\S*$/.test(before)) {
      return { from: word.from, options: pathOptions(meta, before) }
    }
    if (/\?[^#]*$/.test(before) || /&[^#]*$/.test(before)) {
      return { from: word.from, options: toOptions(QUERY_PARAMS, 'property') }
    }
    if (before.includes('{') || before.includes('[')) {
      return { from: word.from, options: bodyOptions(meta, context) }
    }
    return null
  }
}
