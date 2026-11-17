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
