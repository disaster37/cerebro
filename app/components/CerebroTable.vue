<template>
  <div class="w-full space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <UInput
        v-model="globalFilter"
        icon="i-lucide-search"
        size="sm"
        :placeholder="filterPlaceholder || 'Filter rows...'"
        class="w-full sm:w-64"
        data-testid="cerebro-table-filter"
      />
      <UButton
        v-if="globalFilter"
        icon="i-lucide-x"
        size="xs"
        color="neutral"
        variant="ghost"
        aria-label="Clear filter"
        @click="globalFilter = ''"
      />
      <slot name="filters" />
      <div class="ml-auto flex items-center gap-2">
        <span class="text-xs text-neutral-500" data-testid="cerebro-table-count">
          {{ visibleCount }} / {{ data.length }}
        </span>
        <slot name="actions" />
      </div>
    </div>

    <UTable
      ref="tableRef"
      v-model:sorting="sorting"
      v-model:global-filter="globalFilter"
      :data="filteredData"
      :columns="columns"
      :loading="loading"
      :empty="emptyText || 'No rows found.'"
      :get-row-id="rowKey"
      :ui="mergedUi"
      class="w-full"
      @select="onSelect"
    >
      <template v-for="name in cellSlots" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps" />
      </template>
    </UTable>
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, any>">
import { computed, ref, useSlots } from 'vue'
import type { SortingState } from '@tanstack/vue-table'
import { buildSortableColumns, type CerebroColumnDef } from '../composables/useSortableColumns'

const props = defineProps<{
  data: T[]
  columns: CerebroColumnDef<T>[]
  defaultSort?: { id: string; desc: boolean }[]
  filterPlaceholder?: string
  loading?: boolean
  emptyText?: string
  rowKey?: (row: T) => string
  ui?: Record<string, string>
}>()

const emit = defineEmits<{ (e: 'select', row: T): void }>()

const sorting = ref<SortingState>(props.defaultSort ?? [])
const globalFilter = ref('')
const tableRef = ref<any>(null)
const slots = useSlots()

const defaultUi = {
  root: 'relative w-full overflow-x-auto',
  base: 'w-full table-auto',
  th: 'px-2.5 py-2 text-xs font-semibold text-highlighted text-start align-middle whitespace-normal',
  td: 'px-2.5 py-2 text-xs text-muted align-middle whitespace-normal [overflow-wrap:anywhere]',
}

const mergedUi = computed(() => ({ ...defaultUi, ...(props.ui ?? {}) }))
const columns = computed(() => buildSortableColumns(props.columns))
const filteredData = computed(() => props.data)
const cellSlots = computed(() => Object.keys(slots).filter((name) => name.endsWith('-cell')))

const visibleCount = computed(() => {
  const api = tableRef.value?.tableApi
  if (api?.getRowModel) return api.getRowModel().rows.length
  if (!globalFilter.value) return filteredData.value.length
  const query = globalFilter.value.toLowerCase()
  return filteredData.value.filter((row) => JSON.stringify(row).toLowerCase().includes(query)).length
})

function onSelect(row: T) {
  emit('select', row)
}
</script>
