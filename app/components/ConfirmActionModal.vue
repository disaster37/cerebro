<template>
  <UModal v-model:open="isOpen" :title="title">
    <template #header>
      <h3 class="text-base font-semibold">{{ title || 'Confirm action' }}</h3>
    </template>
    <template #body>
      <p class="text-sm text-neutral-600 dark:text-neutral-300">{{ description }}</p>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="cancel">Cancel</UButton>
        <UButton :color="color || 'error'" :loading="loading" @click="confirm">
          {{ confirmLabel || 'Confirm' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  open: boolean
  title?: string
  description?: string
  confirmLabel?: string
  color?: string
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'confirm'): void
}>()

const isOpen = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value),
})

function cancel() {
  emit('update:open', false)
}

function confirm() {
  emit('confirm')
}
</script>
