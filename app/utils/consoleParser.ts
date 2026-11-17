export type ConsoleHttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'HEAD' | 'PATCH'

export interface ParsedConsoleRequest {
  index: number
  method: ConsoleHttpMethod
  rawMethod: string
  path: string
  headerLine: number
  headerFrom: number
  headerTo: number
  from: number
  to: number
  bodyFrom: number | null
  bodyTo: number | null
  rawBody: string
  isNdjsonEndpoint: boolean
}

const HEADER_RE = /^\s*([A-Za-z]+)\s+(\S+)\s*$/
const COMMENT_RE = /^\s*(#|\/\/)/
const NDJSON_RE = /(^|\/)_(bulk|msearch)(\/|$|\?)/

interface LineInfo {
  text: string
  start: number
  end: number
}

function splitLines(doc: string): LineInfo[] {
  const lines: LineInfo[] = []
  let start = 0
  for (let i = 0; i <= doc.length; i++) {
    if (i === doc.length || doc[i] === '\n') {
      lines.push({ text: doc.slice(start, i), start, end: i })
      start = i + 1
    }
  }
  return lines
}

function isNdjsonPath(path: string): boolean {
  return NDJSON_RE.test(path)
}

function updateDepth(line: string, depth: number): number {
  let inString = false
  let escaped = false
  for (const ch of line) {
    if (inString) {
      if (escaped) {
        escaped = false
      } else if (ch === '\\') {
        escaped = true
      } else if (ch === '"') {
        inString = false
      }
      continue
    }
    if (ch === '"') {
      inString = true
    } else if (ch === '{' || ch === '[') {
      depth++
    } else if (ch === '}' || ch === ']') {
      depth--
    }
  }
  return depth
}

export function parseConsoleDocument(doc: string): ParsedConsoleRequest[] {
  const lines = splitLines(doc)
  const requests: ParsedConsoleRequest[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]!
    if (line.text.trim() === '' || COMMENT_RE.test(line.text)) {
      i++
      continue
    }
    const match = HEADER_RE.exec(line.text)
    if (!match) {
      i++
      continue
    }

    const rawMethod = match[1]!
    const path = match[2]!
    const method = rawMethod.toUpperCase() as ConsoleHttpMethod
    const headerLine = i + 1
    const headerFrom = line.start
    const headerTo = line.end
    const ndjson = isNdjsonPath(path)

    let bodyFrom: number | null = null
    let bodyTo: number | null = null
    let rawBody = ''
    let j = i + 1

    if (ndjson) {
      const bodyLines: LineInfo[] = []
      while (j < lines.length) {
        const next = lines[j]!
        if (next.text.trim() === '' || COMMENT_RE.test(next.text) || HEADER_RE.test(next.text)) break
        bodyLines.push(next)
        j++
      }
      if (bodyLines.length) {
        bodyFrom = bodyLines[0]!.start
        bodyTo = bodyLines[bodyLines.length - 1]!.end
        rawBody = doc.slice(bodyFrom, bodyTo)
      }
    } else {
      let depth = 0
      let started = false
      const bodyLines: LineInfo[] = []
      while (j < lines.length) {
        const next = lines[j]!
        if (!started) {
          if (next.text.trim() === '' || COMMENT_RE.test(next.text)) break
          if (HEADER_RE.test(next.text)) break
          const trimmed = next.text.trim()
          if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) break
          started = true
        } else if (HEADER_RE.test(next.text) && depth === 0) {
          break
        }
        bodyLines.push(next)
        depth = updateDepth(next.text, depth)
        j++
        if (started && depth <= 0) break
      }
      if (bodyLines.length) {
        bodyFrom = bodyLines[0]!.start
        bodyTo = bodyLines[bodyLines.length - 1]!.end
        rawBody = doc.slice(bodyFrom, bodyTo)
      }
    }

    requests.push({
      index: requests.length,
      method,
      rawMethod,
      path,
      headerLine,
      headerFrom,
      headerTo,
      from: headerFrom,
      to: bodyTo ?? headerTo,
      bodyFrom,
      bodyTo,
      rawBody,
      isNdjsonEndpoint: ndjson,
    })
    i = j
  }

  return requests
}

export function findRequestAtOffset(requests: ParsedConsoleRequest[], offset: number): ParsedConsoleRequest | null {
  for (const request of requests) {
    if (offset >= request.from && offset <= request.to) return request
  }
  return null
}

export function formatRequestBlockBody(doc: string, req: ParsedConsoleRequest): { from: number; to: number; insert: string } | null {
  if (req.bodyFrom === null || req.bodyTo === null) return null
  const raw = doc.slice(req.bodyFrom, req.bodyTo)
  if (req.isNdjsonEndpoint) {
    const formatted = raw
      .split('\n')
      .map((line) => {
        if (!line.trim()) return line
        try {
          return JSON.stringify(JSON.parse(line))
        } catch {
          return line
        }
      })
      .join('\n')
    return { from: req.bodyFrom, to: req.bodyTo, insert: formatted }
  }
  try {
    return { from: req.bodyFrom, to: req.bodyTo, insert: JSON.stringify(JSON.parse(raw), null, 2) }
  } catch {
    return null
  }
}

export function buildCurlCommand(host: string, req: ParsedConsoleRequest): string {
  const base = (host || '').replace(/\/$/, '')
  const url = `${base}/${req.path.replace(/^\//, '')}`
  const parts = [`curl -X ${req.method} '${url.replace(/'/g, `'\\''`)}'`]
  if (req.rawBody) {
    const contentType = req.isNdjsonEndpoint ? 'application/x-ndjson' : 'application/json'
    parts.push(`-H 'Content-Type: ${contentType}'`)
    parts.push(`-d '${req.rawBody.replace(/'/g, `'\\''`)}'`)
  }
  return parts.join(' \\\n  ')
}
