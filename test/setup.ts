import { vi } from 'vitest'
import { computed, h, reactive, ref } from 'vue'
import { config } from '@vue/test-utils'

const states = new Map<string, any>()

;(globalThis as any).useState = (key: string, init?: () => any) => {
  if (!states.has(key)) states.set(key, ref(init ? init() : undefined))
  return states.get(key)
}

;(globalThis as any).$fetch = vi.fn()
;(globalThis as any).navigateTo = vi.fn()

const colorModeState = reactive({ value: 'dark', preference: 'dark', forced: false })
;(globalThis as any).useColorMode = () => colorModeState
const routeState = reactive<{ query: Record<string, any>; path: string }>({ query: {}, path: '/' })
;(globalThis as any).useRoute = () => routeState

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    props: ['data', 'options'],
    template: '<div class="chart-doughnut" :data-chart="JSON.stringify(data)" :data-options="JSON.stringify(options)" />',
  },
  Bar: {
    props: ['data', 'options'],
    template: '<div class="chart-bar" :data-chart="JSON.stringify(data)" :data-options="JSON.stringify(options)" />',
  },
}))

export const nuxtUiStubs = {
  UCard: {
    props: ['ui'],
    template: `<div class="ucard">
      <div v-if="$slots.header" class="ucard-header"><slot name="header" /></div>
      <div class="ucard-body"><slot /></div>
      <div v-if="$slots.footer" class="ucard-footer"><slot name="footer" /></div>
    </div>`,
  },
  UAlert: {
    props: ['color', 'variant', 'title', 'description', 'close'],
    emits: ['update:open'],
    template: `<div class="ualert" :data-color="color" :data-variant="variant">
      <div v-if="title" class="ualert-title">{{ title }}</div>
      <div v-if="$slots.description" class="ualert-desc"><slot name="description" /></div>
      <div v-else-if="description" class="ualert-desc">{{ description }}</div>
      <button v-if="close" type="button" class="ualert-close" @click="$emit('update:open', false)">x</button>
    </div>`,
  },
  UFormField: {
    props: ['label', 'name', 'description', 'help', 'error', 'required'],
    template: `<div class="uform-field">
      <label v-if="label" class="uform-field-label">{{ label }}</label>
      <slot />
      <p v-if="description" class="uform-field-desc">{{ description }}</p>
    </div>`,
  },
  UInput: {
    inheritAttrs: false,
    props: ['modelValue', 'value', 'size', 'icon', 'placeholder'],
    emits: ['update:modelValue'],
    template: `<input v-bind="$attrs" :value="modelValue ?? value ?? ''" :placeholder="placeholder" @input="$emit('update:modelValue', $event.target.value)" />`,
  },
  UTextarea: {
    inheritAttrs: false,
    props: ['modelValue', 'rows'],
    emits: ['update:modelValue'],
    template: `<textarea v-bind="$attrs" :value="modelValue ?? ''" @input="$emit('update:modelValue', $event.target.value)" />`,
  },
  USelect: {
    inheritAttrs: false,
    props: ['modelValue', 'items', 'placeholder'],
    emits: ['update:modelValue'],
    template: `<select v-bind="$attrs" :value="modelValue ?? ''" @change="$emit('update:modelValue', $event.target.value)">
      <option v-if="placeholder" value="">{{ placeholder }}</option>
      <option
        v-for="item in (items || [])"
        :key="typeof item === 'object' ? item.value : item"
        :value="typeof item === 'object' ? item.value : item"
      >{{ typeof item === 'object' ? item.label : item }}</option>
    </select>`,
  },
  UCheckbox: {
    inheritAttrs: false,
    props: ['modelValue', 'label', 'description', 'id'],
    emits: ['update:modelValue'],
    template: `<div class="ucheckbox">
      <input
        :id="id"
        v-bind="$attrs"
        type="checkbox"
        :checked="Boolean(modelValue)"
        @change="$emit('update:modelValue', $event.target.checked)"
      />
      <label v-if="label" :for="id">{{ label }}</label>
      <p v-if="description" class="ucheckbox-description">{{ description }}</p>
    </div>`,
  },
  USwitch: {
    inheritAttrs: false,
    props: ['modelValue', 'label', 'description'],
    emits: ['update:modelValue'],
    template: `<label class="uswitch">
      <input v-bind="$attrs" type="checkbox" :checked="Boolean(modelValue)" @change="$emit('update:modelValue', $event.target.checked)" />
      <span v-if="label">{{ label }}</span>
    </label>`,
  },
  UButton: {
    props: ['to', 'type', 'color', 'variant', 'size', 'icon', 'block', 'loading', 'disabled'],
    emits: ['click'],
    template: `<a v-if="to" :href="to" class="ubutton"><slot /></a>
      <button v-else :type="type || 'button'" :disabled="disabled || loading" class="ubutton" @click="$emit('click', $event)"><slot /></button>`,
  },
  UBadge: {
    props: ['color', 'variant'],
    template: `<span class="ubadge" :data-color="color"><slot /></span>`,
  },
  UIcon: {
    props: ['name'],
    template: `<span class="uicon" />`,
  },
  UNavigationMenu: {
    props: ['items', 'orientation'],
    template: `<nav class="unav"><slot /></nav>`,
  },
  UDropdownMenu: {
    props: ['items', 'content'],
    template: `<div class="udropdown"><slot /></div>`,
  },
  UProgress: {
    props: ['modelValue', 'size', 'color'],
    template: `<div class="uprogress" :data-value="modelValue" :data-color="color" />`,
  },
  UTooltip: {
    props: ['text'],
    template: `<span class="utooltip" :title="text"><slot /></span>`,
  },
  UModal: {
    props: ['open', 'title', 'modelValue'],
    emits: ['update:open'],
    template: `<div v-if="open || modelValue" class="umodal">
      <div class="umodal-header"><slot name="header" /></div>
      <div class="umodal-body"><slot name="body" /><slot /></div>
      <div class="umodal-footer"><slot name="footer" /></div>
    </div>`,
  },
  UTabs: {
    props: ['items', 'modelValue'],
    emits: ['update:modelValue'],
    template: `<div class="utabs">
      <div class="utabs-list">
        <button v-for="item in (items || [])" :key="item.value || item.label" class="utabs-tab" @click="$emit('update:modelValue', item.value)">
          {{ item.label }}<slot :name="'badge-' + (item.value || item.label)" />
        </button>
      </div>
      <div class="utabs-body"><slot /></div>
    </div>`,
  },
  UColorModeSwitch: {
    props: ['size'],
    template: `<button class="ucolormode-switch" data-testid="color-mode-toggle" @click="toggle">mode</button>`,
    setup() {
      return {
        toggle() {
          colorModeState.value = colorModeState.value === 'dark' ? 'light' : 'dark'
        },
      }
    },
  },
  UColorModeButton: {
    props: ['size'],
    template: `<button class="ucolormode-button" data-testid="color-mode-toggle" @click="toggle">mode</button>`,
    setup() {
      return {
        toggle() {
          colorModeState.value = colorModeState.value === 'dark' ? 'light' : 'dark'
        },
      }
    },
  },
  ClientOnly: {
    template: `<div class="client-only"><slot /></div>`,
  },
  UTable: {
    inheritAttrs: false,
    props: ['data', 'columns', 'sorting', 'globalFilter', 'loading', 'empty', 'ui', 'getRowId'],
    emits: ['update:sorting', 'update:globalFilter', 'select'],
    setup(props: any, { slots, emit, attrs }: any) {
      const isUndefined = (value: any) => value === undefined || value === null
      const rows = computed(() => {
        let data = [...(props.data || [])]
        if (props.globalFilter) {
          const query = String(props.globalFilter).toLowerCase()
          data = data.filter((row: any) => JSON.stringify(row).toLowerCase().includes(query))
        }
        const sorting = props.sorting || []
        if (sorting.length) {
          const sort = sorting[0]
          const col = (props.columns || []).find((c: any) => (c.id || c.accessorKey) === sort.id)
          if (col) {
            data.sort((a: any, b: any) => {
              const av = col.accessorFn ? col.accessorFn(a) : a[col.accessorKey]
              const bv = col.accessorFn ? col.accessorFn(b) : b[col.accessorKey]
              if (isUndefined(av) || isUndefined(bv)) {
                if (isUndefined(av) && isUndefined(bv)) return 0
                const mode = col.sortUndefined
                if (mode === 'first' || mode === -1) return isUndefined(av) ? -1 : 1
                if (mode === 'last' || mode === 1 || mode === undefined) return isUndefined(av) ? 1 : -1
              }
              if (av === bv) return 0
              const cmp = av > bv ? 1 : -1
              return sort.desc ? -cmp : cmp
            })
          }
        }
        return data
      })

      function headerFor(col: any) {
        if (typeof col.header !== 'function') return col.header
        const id = col.id || col.accessorKey
        return col.header({
          column: {
            getIsSorted: () => {
              const sort = (props.sorting || []).find((s: any) => s.id === id)
              return sort ? (sort.desc ? 'desc' : 'asc') : false
            },
            toggleSorting: (desc: boolean) => {
              emit('update:sorting', [{ id, desc }])
            },
          },
        })
      }

      return () => h('table', {
        class: ['utable', attrs.class],
        'data-ui': JSON.stringify(props.ui || {}),
        'data-row-ids': props.getRowId ? JSON.stringify(rows.value.map((row: any) => props.getRowId(row))) : undefined,
      }, [
        h('thead', {}, [
          h('tr', {}, (props.columns || []).map((col: any) => h('th', {
            key: col.id || col.accessorKey,
            class: col.meta?.class?.th,
          }, [headerFor(col)]))),
        ]),
        h('tbody', {}, rows.value.map((item: any, rIdx: number) => h('tr', {
          key: props.getRowId ? props.getRowId(item) : rIdx,
        }, (props.columns || []).map((col: any, cIdx: number) => {
          const name = `${col.id || col.accessorKey}-cell`
          const content = slots[name]
            ? slots[name]({ row: { original: item } })
            : col.cell
              ? col.cell({ row: { original: item } })
              : item[col.accessorKey]
          return h('td', {
            key: col.id || col.accessorKey || cIdx,
            class: col.meta?.class?.td,
          }, [content])
        })))),
      ])
    },
  },
}

config.global.components = {
  ...config.global.components,
  ...nuxtUiStubs,
}

config.global.stubs = {
  ...config.global.stubs,
  NuxtLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
}

export function resetTestStubs() {
  states.clear()
  colorModeState.value = 'dark'
  colorModeState.preference = 'dark'
  routeState.query = {}
  routeState.path = '/'
}

export { routeState }
