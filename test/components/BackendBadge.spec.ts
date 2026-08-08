import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BackendBadge from '@/components/BackendBadge.vue'

describe('BackendBadge', () => {
  it('renders Elasticsearch with version', () => {
    const wrapper = mount(BackendBadge, {
      props: { backendType: 'elasticsearch', backendVersion: '9.4.3' },
    })
    expect(wrapper.text()).toBe('Elasticsearch 9.4.3')
    expect(wrapper.classes()).toContain('text-bg-primary')
  })

  it('renders OpenSearch with version', () => {
    const wrapper = mount(BackendBadge, {
      props: { backendType: 'opensearch', backendVersion: '3.4.0' },
    })
    expect(wrapper.text()).toBe('OpenSearch 3.4.0')
    expect(wrapper.classes()).toContain('badge-opensearch')
  })

  it('renders OpenSearch without version', () => {
    const wrapper = mount(BackendBadge, {
      props: { backendType: 'opensearch' },
    })
    expect(wrapper.text()).toBe('OpenSearch')
  })

  it('renders nothing when backendType is empty', () => {
    const wrapper = mount(BackendBadge, {
      props: { backendType: '', backendVersion: '1.0.0' },
    })
    expect(wrapper.find('[data-testid="backend-badge"]').exists()).toBe(false)
  })
})
