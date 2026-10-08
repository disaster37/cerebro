import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AlertComponent from '@/components/AlertComponent.vue'

const UAlertStub = {
  props: ['color', 'variant', 'description', 'close'],
  emits: ['update:open'],
  template: `<div class="ualert" :data-color="color">
    <span class="ualert-desc">{{ description }}</span>
    <button class="ualert-close" @click="$emit('update:open', false)">x</button>
  </div>`,
}

describe('AlertComponent', () => {
  it('maps danger to error color and emits dismiss on close', async () => {
    const w = mount(AlertComponent, {
      props: { alerts: [{ message: 'boom', type: 'danger' }] },
      global: { components: { UAlert: UAlertStub } },
    })
    expect(w.find('[data-color="error"]').exists()).toBe(true)
    await w.find('.ualert-close').trigger('click')
    expect(w.emitted('dismiss')).toEqual([[0]])
  })
})
