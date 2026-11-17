import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CerebroTable from '@/components/CerebroTable.vue'
import { resetTestStubs } from '../setup'

const rows = [
  { name: 'beta', size: 20 },
  { name: 'alpha', size: 10 },
  { name: 'gamma', size: 30 },
]

const columns = [
  { id: 'name', header: 'Name', accessorKey: 'name' as const },
  { id: 'size', header: 'Size', accessorFn: (row: any) => row.size },
]

describe('CerebroTable', () => {
  beforeEach(() => resetTestStubs())

  it('renders all rows and the result counter', () => {
    const wrapper = mount(CerebroTable, { props: { data: rows, columns } })
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    expect(wrapper.find('[data-testid="cerebro-table-count"]').text()).toBe('3 / 3')
  })

  it('filters rows through the global filter input', async () => {
    const wrapper = mount(CerebroTable, { props: { data: rows, columns } })
    await wrapper.find('[data-testid="cerebro-table-filter"]').setValue('alp')
    expect(wrapper.findAll('tbody tr')).toHaveLength(1)
    expect(wrapper.text()).toContain('alpha')
  })

  it('sorts ascending then descending when the header is clicked', async () => {
    const wrapper = mount(CerebroTable, { props: { data: rows, columns } })
    const header = wrapper.find('[data-testid="sort-size"]')
    expect(header.exists()).toBe(true)

    await header.trigger('click')
    let names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('alpha')
    expect(names[2]).toContain('gamma')

    await header.trigger('click')
    names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('gamma')
    expect(names[2]).toContain('alpha')
  })

  it('applies defaultSort', () => {
    const wrapper = mount(CerebroTable, {
      props: { data: rows, columns, defaultSort: [{ id: 'size', desc: true }] },
    })
    const names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('gamma')
  })
})

describe('CerebroTable full-width layout and sorting', () => {
  beforeEach(() => resetTestStubs())

  it('renders full-width wrapping table ui defaults and forwards class', () => {
    const wrapper = mount(CerebroTable, { props: { data: rows, columns } })
    expect(wrapper.classes()).toContain('w-full')
    const table = wrapper.find('table.utable')
    expect(table.classes()).toContain('w-full')
    const ui = JSON.parse(table.attributes('data-ui') || '{}')
    expect(ui.base).toContain('w-full')
    expect(ui.base).toContain('table-auto')
    expect(ui.td).toContain('[overflow-wrap:anywhere]')
    expect(ui.th).toContain('whitespace-normal')
  })

  it('merges per-page ui overrides with the defaults', () => {
    const wrapper = mount(CerebroTable, {
      props: { data: rows, columns, ui: { td: 'px-1.5 custom-td' } },
    })
    const ui = JSON.parse(wrapper.find('table.utable').attributes('data-ui') || '{}')
    expect(ui.td).toBe('px-1.5 custom-td')
    expect(ui.th).toContain('whitespace-normal')
  })

  it('sorts undefined values last in both directions', async () => {
    const data = [
      { name: 'alpha', size: 10 },
      { name: 'beta', size: undefined },
      { name: 'gamma', size: 30 },
    ]
    const wrapper = mount(CerebroTable, { props: { data, columns } })
    const header = wrapper.find('[data-testid="sort-size"]')

    await header.trigger('click')
    let names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('alpha')
    expect(names[1]).toContain('gamma')
    expect(names[2]).toContain('beta')

    await header.trigger('click')
    names = wrapper.findAll('tbody tr').map((tr) => tr.text())
    expect(names[0]).toContain('gamma')
    expect(names[1]).toContain('alpha')
    expect(names[2]).toContain('beta')
  })

  it('forwards rowKey as getRowId', () => {
    const wrapper = mount(CerebroTable, {
      props: { data: rows, columns, rowKey: (row: any) => `key-${row.name}` },
    })
    expect(JSON.parse(wrapper.find('table.utable').attributes('data-row-ids') || '[]'))
      .toEqual(['key-beta', 'key-alpha', 'key-gamma'])
  })

  it('renders column meta classes on th and td', () => {
    const wrapper = mount(CerebroTable, {
      props: {
        data: rows,
        columns: [
          { id: 'name', header: 'Name', accessorKey: 'name' as const, headerClass: 'w-8', class: 'whitespace-nowrap' },
        ],
      },
    })
    expect(wrapper.find('th').classes()).toContain('w-8')
    expect(wrapper.find('td').classes()).toContain('whitespace-nowrap')
  })
})
