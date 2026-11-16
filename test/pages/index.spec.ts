import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import IndexPage from '@/pages/index.vue'
import { resetTestStubs } from '../setup'

describe('index page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
  })

  it('redirects / to /connect like the original cerebro', async () => {
    mount(IndexPage)
    await flushPromises()

    expect((globalThis as any).navigateTo).toHaveBeenCalledWith('/connect', { replace: true })
  })
})