import { vi } from 'vitest'
import { ref } from 'vue'
import { config } from '@vue/test-utils'

const states = new Map<string, any>()

;(globalThis as any).useState = (key: string, init?: () => any) => {
  if (!states.has(key)) states.set(key, ref(init ? init() : undefined))
  return states.get(key)
}

;(globalThis as any).$fetch = vi.fn()
;(globalThis as any).navigateTo = vi.fn()

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
      <div v-if="description" class="ualert-desc">{{ description }}</div>
      <slot />
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
    props: ['modelValue', 'value', 'size'],
    emits: ['update:modelValue'],
    template: `<input v-bind="$attrs" :value="modelValue ?? value ?? ''" @input="$emit('update:modelValue', $event.target.value)" />`,
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
  UButton: {
    props: ['to', 'type', 'color', 'variant', 'size', 'icon', 'block', 'loading', 'disabled'],
    template: `<a v-if="to" :href="to" class="ubutton"><slot /></a>
      <button v-else :type="type || 'button'" :disabled="disabled || loading" class="ubutton"><slot /></button>`,
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
  UTable: {
    props: ['data', 'columns'],
    template: `<table class="utable">
      <tbody>
        <tr v-for="(item, rIdx) in (data || [])" :key="rIdx">
          <td v-for="(col, cIdx) in (columns || [])" :key="col.id || col.accessorKey || cIdx">
            <slot :name="(col.id || col.accessorKey) + '-cell'" :row="{ original: item }">
              {{ col.cell ? col.cell({ row: { original: item } }) : item[col.accessorKey] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>`,
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
}
