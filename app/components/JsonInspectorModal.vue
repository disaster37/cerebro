<template>
  <UModal v-model:open="isOpen" :title="title">
    <template #header>
      <div class="flex w-full items-center justify-between gap-3">
        <h3 class="text-base font-semibold">{{ title }}</h3>
        <div class="flex items-center gap-2">
          <UInput v-model="search" size="xs" icon="i-lucide-search" placeholder="Search JSON..." />
          <UButton size="xs" color="neutral" variant="ghost" icon="i-lucide-copy" @click="copy">
            Copy
          </UButton>
        </div>
      </div>
    </template>
    <template #body>
      <pre class="max-h-[60vh] overflow-auto rounded-lg bg-neutral-100 p-3 text-xs dark:bg-neutral-900"><code>{{ formatted }}</code></pre>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{
  open: boolean
  title?: string
  data: any
}>()

const emit = defineEmits<{ (e: 'update:open', value: boolean): void }>()

const search = ref('')

const isOpen = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
})

const formatted = computed(() => {
  let text: string
  try {
    text = typeof props.data === 'string' ? props.data : JSON.stringify(props.data, null, 2)
  } catch {
    text = String(props.data)
  }
  if (!search.value) return text
  return text
    .split('\n')
    .filter((line) => line.toLowerCase().includes(search.value.toLowerCase()))
    .join('\n')
})

async function copy() {
  try {
    await navigator.clipboard.writeText(formatted.value)
  } catch {
    // Clipboard may be unavailable (insecure context); ignore.
  }
}
</script>
