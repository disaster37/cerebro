import type { Diagnostic } from '@codemirror/lint'
import { parseConsoleDocument, type ParsedConsoleRequest } from './consoleParser'

const ALLOWED_METHODS = new Set(['GET', 'POST', 'PUT', 'DELETE', 'HEAD', 'PATCH'])
const HEADER_RE = /^\s*([A-Za-z]+)\s+(\S+)\s*$/
const METHOD_ONLY_RE = /^\s*([A-Za-z]+)\s*$/
const COMMENT_RE = /^\s*(#|\/\/)/

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

function jsonErrorOffset(message: string, raw: string): number {
  const position = /position (\d+)/.exec(message)
  if (position) return Math.min(Number(position[1]), raw.length)
  const lineCol = /line (\d+) column (\d+)/.exec(message)
  if (lineCol) {
    const line = Number(lineCol[1])
    const column = Number(lineCol[2])
    const lines = raw.split('\n')
    let offset = 0
    for (let i = 0; i < line - 1 && i < lines.length; i++) offset += lines[i]!.length + 1
    return Math.min(offset + column - 1, raw.length)
  }
  return 0
}

function validateJsonBody(doc: string, req: ParsedConsoleRequest, diagnostics: Diagnostic[]) {
  if (req.bodyFrom === null || req.bodyTo === null) return
  const raw = doc.slice(req.bodyFrom, req.bodyTo)

  if (req.isNdjsonEndpoint) {
    let offset = req.bodyFrom
    for (const line of raw.split('\n')) {
      const trimmed = line.trim()
      if (trimmed) {
        try {
          const parsed = JSON.parse(trimmed)
          if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
            diagnostics.push({
              from: offset,
              to: offset + line.length,
              severity: 'error',
              message: 'Each NDJSON line must be a JSON object',
            })
          }
        } catch (err: any) {
          diagnostics.push({
            from: offset,
            to: offset + line.length,
            severity: 'error',
            message: `Invalid NDJSON line: ${err.message}`,
          })
        }
      }
      offset += line.length + 1
    }
    if (req.method === 'HEAD') {
      diagnostics.push({
        from: req.bodyFrom,
        to: req.bodyTo,
        severity: 'warning',
        message: 'HEAD requests should not include a body',
      })
    }
    return
  }

  try {
    JSON.parse(raw)
  } catch (err: any) {
    const relative = jsonErrorOffset(err.message || '', raw)
    const from = req.bodyFrom + relative
    diagnostics.push({
      from,
      to: Math.min(from + 1, req.bodyTo),
      severity: 'error',
      message: err.message || 'Invalid JSON',
    })
  }
}

export function validateConsoleDocument(doc: string): Diagnostic[] {
  const diagnostics: Diagnostic[] = []
  const requests = parseConsoleDocument(doc)
  const lines = splitLines(doc)

  const covered = new Set<number>()
  for (const request of requests) {
    const endLine = request.bodyTo === null
      ? request.headerLine
      : lines.findIndex((line) => line.end >= request.bodyTo!) + 1
    for (let line = request.headerLine; line <= endLine; line++) covered.add(line)
  }

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]!
    const lineNumber = i + 1
    if (line.text.trim() === '' || COMMENT_RE.test(line.text)) continue

    const methodOnly = METHOD_ONLY_RE.exec(line.text)
    if (methodOnly) {
      diagnostics.push({
        from: line.start,
        to: line.end,
        severity: 'error',
        message: 'Missing request path',
      })
      continue
    }

    const header = HEADER_RE.exec(line.text)
    if (header) {
      const rawMethod = header[1]!
      if (!ALLOWED_METHODS.has(rawMethod.toUpperCase())) {
        const methodStart = line.start + line.text.indexOf(rawMethod)
        diagnostics.push({
          from: methodStart,
          to: methodStart + rawMethod.length,
          severity: 'error',
          message: `Unrecognized HTTP method "${rawMethod}"`,
        })
      }
      const request = requests.find((r) => r.headerLine === lineNumber)
      if (request) validateJsonBody(doc, request, diagnostics)
      continue
    }

    if (!covered.has(lineNumber)) {
      diagnostics.push({
        from: line.start,
        to: line.end,
        severity: 'error',
        message: 'Orphan text: expected a "METHOD path" request header',
      })
    }
  }

  return diagnostics
}
