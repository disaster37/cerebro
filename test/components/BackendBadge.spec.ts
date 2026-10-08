import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BackendBadge from '@/components/BackendBadge.vue'

const UBadgeStub = {
  props: ['color', 'variant'],
  template: '<span class="ubadge" data-testid="backend-badge" :data-color="color"><slot /></span>',
}

const mountBadge = (props: Record<string, unknown>) =>
  mount(BackendBadge, { props, global: { components: { UBadge: UBadgeStub } } })

describe('BackendBadge', () => {
  it('renders Elasticsearch with version', () => {
    const w = mountBadge({ backendType: 'elasticsearch', backendVersion: '9.4.3' })
    expect(w.text()).toBe('Elasticsearch 9.4.3')
    expect(w.find('[data-color="primary"]').exists()).toBe(true)
  })
  it('renders OpenSearch with version', () => {
    const w = mountBadge({ backendType: 'opensearch', backendVersion: '3.4.0' })
    expect(w.text()).toBe('OpenSearch 3.4.0')
    expect(w.find('[data-color="secondary"]').exists()).toBe(true)
  })
  it('renders OpenSearch without version', () => {
    const w = mountBadge({ backendType: 'opensearch' })
    expect(w.text()).toBe('OpenSearch')
  })
  it('renders nothing when backendType is empty', () => {
    const w = mountBadge({ backendType: '', backendVersion: '1.0.0' })
    expect(w.find('[data-testid="backend-badge"]').exists()).toBe(false)
  })
})
