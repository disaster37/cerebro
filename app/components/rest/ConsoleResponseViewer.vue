<template>
  <div class="flex h-full flex-col">
    <div class="flex items-center gap-2 border-b border-neutral-200 px-2 py-1.5 dark:border-neutral-800">
      <UBadge :color="statusColor" variant="soft">{{ statusLabel }}</UBadge>
      <span class="text-xs text-neutral-500">{{ duration }} ms</span>
      <span class="text-xs text-neutral-500">{{ formatBytes(size) }}</span>
      <UButton class="ml-auto" size="xs" color="neutral" variant="ghost" icon="i-lucide-copy" @click="copy">
        Copy JSON
      </UButton>
    </div>
    <div ref="host" class="cm-console-host flex-1" data-testid="console-response" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Compartment, EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { foldGutter } from '@codemirror/language'
import { json } from '@codemirror/lang-json'
import { searchKeymap } from '@codemirror/search'
import { formatBytes } from '../../utils/format'

const props = defineProps<{
  status: number
  duration: number
  size: number
  body: any
}>()

const host = ref<HTMLDivElement | null>(null)
const colorMode = useColorMode()
const themeCompartment = new Compartment()
let view: EditorView | null = null

const statusLabel = computed(() => {
  if (!props.status) return 'No response'
  return `${props.status} ${props.status < 300 ? 'OK' : props.status < 500 ? 'Client Error' : 'Server Error'}`
})

const statusColor = computed(() => {
  if (!props.status) return 'neutral'
  if (props.status < 300) return 'success'
  if (props.status < 500) return 'warning'
  return 'error'
})

function text() {
  try {
    return typeof props.body === 'string' ? props.body : JSON.stringify(props.body, null, 2)
  } catch {
    return String(props.body)
  }
}

function darkTheme() {
  return EditorView.theme({
    '&': { backgroundColor: '#09090b', color: '#e4e4e7' },
    '.cm-gutters': { backgroundColor: '#09090b', color: '#71717a', border: 'none' },
  }, { dark: true })
}

function lightTheme() {
  return EditorView.theme({
    '&': { backgroundColor: '#ffffff', color: '#18181b' },
    '.cm-gutters': { backgroundColor: '#fafafa', color: '#71717a', border: 'none' },
  })
}

onMounted(() => {
  if (!host.value) return
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: text(),
      extensions: [
        lineNumbers(),
        foldGutter(),
        history(),
        json(),
        keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap]),
        EditorView.editable.of(false),
        EditorState.readOnly.of(true),
        themeCompartment.of(colorMode.value === 'dark' ? darkTheme() : lightTheme()),
      ],
    }),
  })
})

watch(() => props.body, () => {
  if (!view) return
  view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text() } })
})

watch(() => colorMode.value, () => {
  if (!view) return
  view.dispatch({ effects: themeCompartment.reconfigure(colorMode.value === 'dark' ? darkTheme() : lightTheme()) })
})

onBeforeUnmount(() => {
  view?.destroy()
  view = null
})

async function copy() {
  try {
    await navigator.clipboard.writeText(text())
  } catch {
    // Clipboard may be unavailable; ignore.
  }
}
</script>
