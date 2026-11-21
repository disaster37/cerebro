import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'

export interface CerebroColumnDef<T> {
  id: string
  header: string
  accessorKey?: keyof T & string
  accessorFn?: (row: T) => string | number | boolean | null | undefined
  sortable?: boolean
  sortUndefined?: 'first' | 'last' | false | -1 | 1
  cell?: TableColumn<T>['cell']
  meta?: TableColumn<T>['meta']
  class?: string
  headerClass?: string
}

function columnMeta<T>(def: CerebroColumnDef<T>): TableColumn<T>['meta'] {
  const existing = (def.meta as { class?: Record<string, string> } | undefined)?.class
  if (!def.meta && !def.class && !def.headerClass) return undefined
  return {
    ...def.meta,
    class: {
      ...existing,
      ...(def.headerClass ? { th: def.headerClass } : {}),
      ...(def.class ? { td: def.class } : {}),
    },
  } as TableColumn<T>['meta']
}

export function buildSortableColumns<T>(defs: CerebroColumnDef<T>[]): TableColumn<T>[] {
  return defs.map((def) => {
    const column: TableColumn<T> = {
      id: def.id,
      accessorKey: def.accessorKey,
      accessorFn: def.accessorFn,
      cell: def.cell,
      meta: columnMeta(def),
      sortUndefined: def.sortUndefined ?? 'last',
    } as TableColumn<T>

    if (def.sortable === false) {
      column.enableSorting = false
      column.header = def.header
      return column
    }

    column.enableSorting = true
    column.sortingFn = 'auto'
    column.header = ({ column: tableColumn }: any) => {
      const sorted = tableColumn.getIsSorted()
      const icon = sorted === 'asc'
        ? 'i-lucide-arrow-up'
        : sorted === 'desc'
          ? 'i-lucide-arrow-down'
          : 'i-lucide-arrow-up-down'
      return h(
        resolveComponent('UButton'),
        {
          variant: 'ghost',
          color: 'neutral',
          size: 'xs',
          class: 'px-0 h-auto text-left whitespace-normal font-semibold',
          'data-testid': `sort-${def.id}`,
          onClick: () => tableColumn.toggleSorting(tableColumn.getIsSorted() === 'asc'),
        },
        () => [
          h('span', { class: 'mr-1 leading-tight' }, def.header),
          h('span', { class: `${icon} shrink-0`, 'aria-hidden': 'true' }),
        ],
      )
    }
    return column
  })
}
