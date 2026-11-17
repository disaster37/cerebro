<template>
  <div ref="host" class="cm-console-host" data-testid="console-editor" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState, Compartment, RangeSetBuilder, type Range } from '@codemirror/state'
import {
  Decoration,
  EditorView,
  GutterMarker,
  WidgetType,
  crosshairCursor,
  drawSelection,
  dropCursor,
  gutter,
  highlightActiveLine,
  highlightActiveLineGutter,
  keymap,
  lineNumbers,
  rectangularSelection,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import { bracketMatching, defaultHighlightStyle, foldGutter, indentOnInput, syntaxHighlighting } from '@codemirror/language'
import { json } from '@codemirror/lang-json'
import { acceptCompletion, autocompletion, closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { lintGutter, linter } from '@codemirror/lint'
import { highlightSelectionMatches, searchKeymap } from '@codemirror/search'
import type { RestMetadata } from '../../types/cerebro'
import { createConsoleCompletionSource } from '../../utils/consoleAutocomplete'
import { validateConsoleDocument } from '../../utils/consoleLinter'
import {
  findRequestAtOffset,
  formatRequestBlockBody,
  parseConsoleDocument,
  type ParsedConsoleRequest,
} from '../../utils/consoleParser'

const props = defineProps<{
  modelValue: string
  meta: RestMetadata
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'send', request: ParsedConsoleRequest): void
  (e: 'copy-curl', request: ParsedConsoleRequest): void
  (e: 'active-change', request: ParsedConsoleRequest): void
}>()

const host = ref<HTMLDivElement | null>(null)
const colorMode = useColorMode()
const themeCompartment = new Compartment()
let view: EditorView | null = null

class PlayMarker extends GutterMarker {
  constructor(private readonly line: number) {
    super()
  }

  toDOM() {
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'cm-play-gutter'
    button.textContent = '▶'
    button.title = 'Send request'
    button.addEventListener('click', (event) => {
      event.preventDefault()
      sendAtLine(this.line)
    })
    return button
  }
}

class RequestHeaderActionsWidget extends WidgetType {
  constructor(
    private readonly request: ParsedConsoleRequest,
    private readonly onSend: (request: ParsedConsoleRequest) => void,
    private readonly onCopy: (request: ParsedConsoleRequest) => void,
    private readonly onFormat: (request: ParsedConsoleRequest) => void,
  ) {
    super()
  }

  eq(other: RequestHeaderActionsWidget) {
    return other.request.from === this.request.from && other.request.to === this.request.to
  }

  toDOM() {
    const span = document.createElement('span')
    span.className = 'cm-header-actions'

    const send = document.createElement('button')
    send.type = 'button'
    send.textContent = '▶ Send'
    send.title = 'Send request'
    send.addEventListener('mousedown', (event) => {
      event.preventDefault()
      this.onSend(this.request)
    })
    span.appendChild(send)

    const copy = document.createElement('button')
    copy.type = 'button'
    copy.textContent = 'Copy as cURL'
    copy.title = 'Copy as cURL'
    copy.addEventListener('mousedown', (event) => {
      event.preventDefault()
      this.onCopy(this.request)
      copy.textContent = 'Copied!'
      setTimeout(() => { copy.textContent = 'Copy as cURL' }, 1500)
    })
    span.appendChild(copy)

    if (this.request.bodyFrom !== null) {
      const indent = document.createElement('button')
      indent.type = 'button'
      indent.textContent = 'Auto-indent'
      indent.title = 'Auto-indent request body'
      indent.addEventListener('mousedown', (event) => {
        event.preventDefault()
        this.onFormat(this.request)
      })
      span.appendChild(indent)
    }

    return span
  }

  ignoreEvent() {
    return false
  }
}

function playMarkers(editorView: EditorView) {
  const builder = new RangeSetBuilder<GutterMarker>()
  const requests = parseConsoleDocument(editorView.state.doc.toString())
  for (const request of requests) {
    const line = editorView.state.doc.line(request.headerLine)
    builder.add(line.from, line.from, new PlayMarker(request.headerLine))
  }
  return builder.finish()
}

// buildConsoleDecorations receives an EditorState (Facet.compute passes the
// state, not the view). It highlights the active request block with line
// decorations and attaches the hover action widget to every request header.
function buildConsoleDecorations(state: EditorState) {
  const requests = parseConsoleDocument(state.doc.toString())
  const decorations: Range<Decoration>[] = []

  const active = findRequestAtOffset(requests, state.selection.main.head)
  if (active) {
    const startLine = state.doc.lineAt(active.from).number
    const endLine = state.doc.lineAt(active.to).number
    for (let n = startLine; n <= endLine; n++) {
      const line = state.doc.line(n)
      decorations.push(Decoration.line({ class: 'cm-active-request-line' }).range(line.from))
    }
  }

  for (const request of requests) {
    const headerLine = state.doc.line(request.headerLine)
    decorations.push(Decoration.widget({
      widget: new RequestHeaderActionsWidget(
        request,
        (r) => emit('send', r),
        (r) => emit('copy-curl', r),
        (r) => formatRequest(r),
      ),
      side: 1,
    }).range(headerLine.to))
  }

  return Decoration.set(decorations, true)
}

function darkTheme() {
  return EditorView.theme({
    '&': { backgroundColor: '#09090b', color: '#e4e4e7' },
    '.cm-content': { caretColor: '#f4f4f5' },
    '.cm-gutters': { backgroundColor: '#09090b', color: '#71717a', border: 'none' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,0.04)' },
    '.cm-activeLineGutter': { backgroundColor: 'rgba(255,255,255,0.06)' },
    '.cm-selectionBackground, ::selection': { backgroundColor: 'rgba(56,189,248,0.25)' },
    '.cm-tooltip.cm-tooltip-autocomplete': {
      backgroundColor: '#18181b',
      color: '#e4e4e7',
      border: '1px solid #3f3f46',
    },
    '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
      backgroundColor: 'rgba(56,189,248,0.2)',
      color: '#f4f4f5',
    },
  }, { dark: true })
}

function lightTheme() {
  return EditorView.theme({
    '&': { backgroundColor: '#ffffff', color: '#18181b' },
    '.cm-content': { caretColor: '#18181b' },
    '.cm-gutters': { backgroundColor: '#fafafa', color: '#71717a', border: 'none' },
    '.cm-activeLine': { backgroundColor: 'rgba(0,0,0,0.03)' },
    '.cm-activeLineGutter': { backgroundColor: 'rgba(0,0,0,0.05)' },
  })
}

function currentRequest(): ParsedConsoleRequest | null {
  if (!view) return null
  const requests = parseConsoleDocument(view.state.doc.toString())
  return findRequestAtOffset(requests, view.state.selection.main.head) || requests[0] || null
}

function sendAtLine(line: number) {
  if (!view) return
  const requests = parseConsoleDocument(view.state.doc.toString())
  const request = requests.find((r) => r.headerLine === line)
  if (request) emit('send', request)
}

function sendActive() {
  const request = currentRequest()
  if (request) emit('send', request)
  return true
}

function formatRequest(request: ParsedConsoleRequest) {
  if (!view) return false
  const formatted = formatRequestBlockBody(view.state.doc.toString(), request)
  if (!formatted) return false
  view.dispatch({ changes: { from: formatted.from, to: formatted.to, insert: formatted.insert } })
  return true
}

function formatActive() {
  const request = currentRequest()
  if (!request) return false
  return formatRequest(request)
}

defineExpose({ sendActive, formatActive })

onMounted(() => {
  if (!host.value) return
  const metaRef = computed(() => props.meta)
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        lineNumbers(),
        highlightActiveLineGutter(),
        highlightActiveLine(),
        foldGutter(),
        drawSelection(),
        dropCursor(),
        rectangularSelection(),
        crosshairCursor(),
        history(),
        indentOnInput(),
        bracketMatching(),
        closeBrackets(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        json(),
        linter(validateConsoleDocument),
        lintGutter(),
        autocompletion({ activateOnTyping: true, override: [createConsoleCompletionSource(metaRef)] }),
        highlightSelectionMatches(),
        gutter({ class: 'cm-play-gutter', markers: playMarkers }),
        EditorView.decorations.compute(['doc', 'selection'], buildConsoleDecorations),
        keymap.of([
          { key: 'Mod-Enter', run: () => sendActive() },
          { key: 'Mod-Shift-f', run: () => formatActive() },
          { key: 'Tab', run: acceptCompletion },
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...historyKeymap,
          ...searchKeymap,
          indentWithTab,
        ]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) emit('update:modelValue', update.state.doc.toString())
          if (update.docChanged || update.selectionSet) {
            const requests = parseConsoleDocument(update.state.doc.toString())
            const request = findRequestAtOffset(requests, update.state.selection.main.head) || requests[0] || null
            if (request) emit('active-change', request)
          }
        }),
        themeCompartment.of(colorMode.value === 'dark' ? darkTheme() : lightTheme()),
      ],
    }),
  })
})

watch(() => props.modelValue, (value) => {
  if (!view) return
  if (value !== view.state.doc.toString()) {
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: value } })
  }
})

watch(() => colorMode.value, () => {
  if (!view) return
  view.dispatch({ effects: themeCompartment.reconfigure(colorMode.value === 'dark' ? darkTheme() : lightTheme()) })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})
</script>
