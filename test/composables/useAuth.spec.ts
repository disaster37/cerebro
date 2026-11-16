import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LoginPage from '@/pages/login.vue'
import {
  consumeRedirectCookie,
  sanitizeRedirectPath,
  useAuth,
} from '@/composables/useAuth'
import { defaultClusterConnection } from '@/composables/useAPI'
import { resetTestStubs } from '../setup'

function stubOpaqueRedirect() {
  const fetchMock = vi.fn().mockResolvedValue({
    status: 0,
    redirected: false,
    type: 'opaqueredirect',
  })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('useAuth', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
    document.cookie = 'redirect=; Max-Age=0; path=/'
  })

  it('checkSession sets authEnabled and user from GET /auth/me', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: { authEnabled: true, username: 'alice' },
    })

    const { checkSession, user, authEnabled } = useAuth()
    const result = await checkSession()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/auth/me', expect.objectContaining({
      method: 'GET',
    }))
    expect(result).toBe('alice')
    expect(user.value).toBe('alice')
    expect(authEnabled.value).toBe(true)
  })

  it('login verifies an opaqueredirect response via GET /auth/me', async () => {
    const fetchMock = stubOpaqueRedirect()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: { authEnabled: true, username: 'alice' },
    })

    const { login } = useAuth()
    const ok = await login('alice', 'pw')

    expect(ok).toBe(true)
    expect(fetchMock).toHaveBeenCalledWith('/auth/login', expect.objectContaining({
      method: 'POST',
      redirect: 'manual',
    }))
  })

  it('login returns false when the session is not established', async () => {
    stubOpaqueRedirect()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: { authEnabled: true },
    })

    const { login } = useAuth()
    expect(await login('alice', 'wrong')).toBe(false)
  })

  it('login fails closed when session verification errors', async () => {
    stubOpaqueRedirect()
    ;(globalThis as any).$fetch.mockRejectedValueOnce(new Error('network down'))

    const { login } = useAuth()
    expect(await login('alice', 'pw')).toBe(false)
  })

  it('logout clears session and cluster state and navigates to /login', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { logout, user } = useAuth()
    user.value = 'alice'
    ;(globalThis as any).useState('clusterConnection').value = {
      host: 'https://es:9200',
      username: 'elastic',
      password: 'secret',
      trustCertificate: true,
    }
    ;(globalThis as any).useState('clusterName').value = 'prod'
    ;(globalThis as any).useState('trustCertificate').value = true
    ;(globalThis as any).useState('backendType').value = 'opensearch'
    ;(globalThis as any).useState('backendVersion').value = '3.4.0'

    await logout()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/auth/logout', expect.objectContaining({
      method: 'POST',
    }))
    expect(user.value).toBe('')
    expect((globalThis as any).useState('clusterConnection').value).toEqual(defaultClusterConnection())
    expect((globalThis as any).useState('clusterName').value).toBe('')
    expect((globalThis as any).useState('trustCertificate').value).toBe(false)
    expect((globalThis as any).useState('backendType').value).toBe('')
    expect((globalThis as any).useState('backendVersion').value).toBe('')
    expect((globalThis as any).navigateTo).toHaveBeenCalledWith('/login')
  })

  it('sanitizeRedirectPath blocks open redirects and control characters', () => {
    expect(sanitizeRedirectPath('/overview')).toBe('/overview')
    expect(sanitizeRedirectPath('%2Foverview')).toBe('/overview')
    expect(sanitizeRedirectPath('//evil.com')).toBe('/')
    expect(sanitizeRedirectPath('/\\evil.com')).toBe('/')
    expect(sanitizeRedirectPath('/foo\nbar')).toBe('/')
    expect(sanitizeRedirectPath(undefined)).toBe('/')
    expect(sanitizeRedirectPath('')).toBe('/')
  })

  it('consumeRedirectCookie returns and clears the sanitized redirect', () => {
    document.cookie = 'redirect=%2Foverview'
    expect(consumeRedirectCookie()).toBe('/overview')
    expect(document.cookie).not.toContain('redirect=%2Foverview')
  })

  it('consumeRedirectCookie returns / when no cookie is present', () => {
    expect(consumeRedirectCookie()).toBe('/')
  })
})

describe('LoginPage', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
    ;(globalThis as any).navigateTo.mockReset()
    document.cookie = 'redirect=; Max-Age=0; path=/'
  })

  it('consumes the redirect cookie and navigates after a valid login', async () => {
    stubOpaqueRedirect()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: { authEnabled: true, username: 'alice' },
    })
    document.cookie = 'redirect=%2Foverview'

    const wrapper = mount(LoginPage)
    await wrapper.find('[data-testid="login-username"]').setValue('alice')
    await wrapper.find('[data-testid="login-password"]').setValue('pw')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect((globalThis as any).navigateTo).toHaveBeenCalledWith('/overview')
  })

  it('renders an inline error and does not navigate on invalid login', async () => {
    stubOpaqueRedirect()
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: { authEnabled: true },
    })

    const wrapper = mount(LoginPage)
    await wrapper.find('[data-testid="login-username"]').setValue('alice')
    await wrapper.find('[data-testid="login-password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const alert = wrapper.find('[data-testid="login-error"]')
    expect(alert.exists()).toBe(true)
    expect(alert.attributes('data-color')).toBe('error')
    expect(alert.text()).toContain('Invalid username or password')
    expect((globalThis as any).navigateTo).not.toHaveBeenCalled()
  })
})
