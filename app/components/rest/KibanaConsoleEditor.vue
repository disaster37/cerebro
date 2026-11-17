<template>
  <div ref="host" class="cm-console-host" data-testid="console-editor" />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorState, Compartment, RangeSetBuilder } from '@codemirror/state'
import {
  Decoration,
  EditorView,
  GutterMarker,
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
import { autocompletion } from '@codemirror/autocomplete'
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

function playMarkers(editorView: EditorView) {
  const builder = new RangeSetBuilder<GutterMarker>()
  const requests = parseConsoleDocument(editorView.state.doc.toString())
  for (const request of requests) {
    const line = editorView.state.doc.line(request.headerLine)
    builder.add(line.from, line.from, new PlayMarker(request.headerLine))
  }
  return builder.finish()
}

function activeRequestDecorations(editorView: EditorView) {
  const requests = parseConsoleDocument(editorView.state.doc.toString())
  const request = findRequestAtOffset(requests, editorView.state.selection.main.head)
  if (!request) return Decoration.none
  return Decoration.set([
    Decoration.mark({ class: 'cm-active-request' }).range(request.from, request.to),
  ])
}

function darkTheme() {
  return EditorView.theme({
    '&': { backgroundColor: '#09090b', color: '#e4e4e7' },
    '.cm-content': { caretColor: '#f4f4f5' },
    '.cm-gutters': { backgroundColor: '#09090b', color: '#71717a', border: 'none' },
    '.cm-activeLine': { backgroundColor: 'rgba(255,255,255,0.04)' },
    '.cm-activeLineGutter': { backgroundColor: 'rgba(255,255,255,0.06)' },
    '.cm-selectionBackground, ::selection': { backgroundColor: 'rgba(56,189,248,0.25)' },
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

function formatActive() {
  if (!view) return false
  const request = currentRequest()
  if (!request) return false
  const formatted = formatRequestBlockBody(view.state.doc.toString(), request)
  if (!formatted) return false
  view.dispatch({ changes: { from: formatted.from, to: formatted.to, insert: formatted.insert } })
  return true
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
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        json(),
        linter(validateConsoleDocument),
        lintGutter(),
        autocompletion({ override: [createConsoleCompletionSource(metaRef)] }),
        highlightSelectionMatches(),
        gutter({ class: 'cm-play-gutter', markers: playMarkers }),
        EditorView.decorations.compute(['doc', 'selection'], activeRequestDecorations),
        keymap.of([
          { key: 'Mod-Enter', run: () => sendActive() },
          { key: 'Mod-Shift-f', run: () => formatActive() },
          ...defaultKeymap,
          ...historyKeymap,
          ...searchKeymap,
          indentWithTab,
        ]),
        EditorView.updateListener.of((update) => {
          if (update.docChanged) emit('update:modelValue', update.state.doc.toString())
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
