import { h, resolveComponent } from 'vue'
import type { TableColumn } from '@nuxt/ui'

export interface CerebroColumnDef<T> {
  id: string
  header: string
  accessorKey?: keyof T & string
  accessorFn?: (row: T) => string | number | boolean | null | undefined
  sortable?: boolean
  cell?: TableColumn<T>['cell']
  meta?: TableColumn<T>['meta']
}

export function buildSortableColumns<T>(defs: CerebroColumnDef<T>[]): TableColumn<T>[] {
  return defs.map((def) => {
    const column: TableColumn<T> = {
      id: def.id,
      accessorKey: def.accessorKey,
      accessorFn: def.accessorFn,
      cell: def.cell,
      meta: def.meta,
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
          class: 'px-0',
          'data-testid': `sort-${def.id}`,
          onClick: () => tableColumn.toggleSorting(tableColumn.getIsSorted() === 'asc'),
        },
        () => [
          h('span', { class: 'mr-1' }, def.header),
          h('span', { class: icon, 'aria-hidden': 'true' }),
        ],
      )
    }
    return column
  })
}
