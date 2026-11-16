import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import NavbarComponent from '@/components/NavbarComponent.vue'
import { resetTestStubs } from '../setup'

function mountNavbar() {
  return mount(NavbarComponent)
}

function stubAuthMe(body: Record<string, any>) {
  ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body })
}

function setConnection(username: string) {
  ;(globalThis as any).useState('clusterConnection').value = {
    host: 'https://es:9200',
    username,
    password: username ? 'secret' : '',
    trustCertificate: false,
  }
}

describe('NavbarComponent', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('hides Login, Logout and user when auth is disabled and no connection exists', async () => {
    stubAuthMe({ authEnabled: false })

    const wrapper = mountNavbar()
    await flushPromises()

    expect(wrapper.find('[data-testid="navbar-login"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="navbar-logout"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="navbar-user"]').exists()).toBe(false)
  })

  it('shows the cluster username and hides auth actions when auth is disabled', async () => {
    stubAuthMe({ authEnabled: false })
    setConnection('elastic')

    const wrapper = mountNavbar()
    await flushPromises()

    expect(wrapper.find('[data-testid="navbar-user"]').text()).toBe('elastic')
    expect(wrapper.find('[data-testid="navbar-login"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="navbar-logout"]').exists()).toBe(false)
  })

  it('shows Login and hides Logout when auth is enabled and logged out', async () => {
    stubAuthMe({ authEnabled: true })

    const wrapper = mountNavbar()
    await flushPromises()

    expect(wrapper.find('[data-testid="navbar-login"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="navbar-logout"]').exists()).toBe(false)
  })

  it('prefers the cluster username over the Cerebro session user', async () => {
    stubAuthMe({ authEnabled: true, username: 'cerebro-admin' })
    setConnection('elastic')

    const wrapper = mountNavbar()
    await flushPromises()

    expect(wrapper.find('[data-testid="navbar-user"]').text()).toBe('elastic')
    expect(wrapper.find('[data-testid="navbar-logout"]').exists()).toBe(true)
  })

  it('falls back to the Cerebro session user when no cluster username exists', async () => {
    stubAuthMe({ authEnabled: true, username: 'cerebro-admin' })
    setConnection('')

    const wrapper = mountNavbar()
    await flushPromises()

    expect(wrapper.find('[data-testid="navbar-user"]').text()).toBe('cerebro-admin')
    expect(wrapper.find('[data-testid="navbar-logout"]').exists()).toBe(true)
  })

  it('logs out and navigates to /login on click', async () => {
    ;(globalThis as any).$fetch.mockImplementation((url: string) => {
      if (url === '/auth/me') {
        return Promise.resolve({ status: 200, body: { authEnabled: true, username: 'cerebro-admin' } })
      }
      return Promise.resolve({ status: 200, body: {} })
    })

    const wrapper = mountNavbar()
    await flushPromises()

    await wrapper.find('[data-testid="navbar-logout"]').trigger('click')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({
      method: 'POST',
    }))
    expect((globalThis as any).navigateTo).toHaveBeenCalledWith('/login')
  })
})
