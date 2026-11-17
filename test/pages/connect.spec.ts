import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ConnectPage from '@/pages/connect.vue'
import { resetTestStubs } from '../setup'

async function mountPage(overrides: Record<string, any> = {}) {
  ;(globalThis as any).$fetch.mockImplementation((url: string) => {
    if (url === '/connect/hosts') {
      return Promise.resolve({ status: 200, body: ['http://es:9200'] })
    }
    if (url in overrides) {
      return Promise.resolve(overrides[url])
    }
    return Promise.resolve({ status: 200, body: {} })
  })
  const wrapper = mount(ConnectPage)
  await flushPromises()
  return wrapper
}

describe('connect page', () => {
  beforeEach(() => {
    resetTestStubs()
    vi.clearAllMocks()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('renders a centered card with labels, warning text, and an unchecked trust checkbox', async () => {
    const wrapper = await mountPage()

    const card = wrapper.find('.connect-card')
    expect(card.exists()).toBe(true)
    expect(card.classes()).toContain('mx-auto')

    const text = wrapper.text()
    for (const label of [
      'Configured Host',
      'Custom Host URL',
      'Username',
      'Password',
      'Trust self-signed certificate',
    ]) {
      expect(text).toContain(label)
    }

    for (const testId of [
      'connect-host',
      'connect-custom-host',
      'connect-username',
      'connect-password',
      'connect-trust-cert',
      'connect-submit',
    ]) {
      expect(wrapper.find(`[data-testid="${testId}"]`).exists()).toBe(true)
    }

    const checkbox = wrapper.find('[data-testid="connect-trust-cert"]')
    expect((checkbox.element as HTMLInputElement).checked).toBe(false)
    expect(text).toContain('Warning: Disables TLS certificate verification')
  })

  it('disables submit and does not call /connect when no host is provided', async () => {
    const wrapper = await mountPage()

    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const connectCalls = (globalThis as any).$fetch.mock.calls.filter(
      (call: any[]) => call[0] === '/connect',
    )
    expect(connectCalls).toHaveLength(0)
  })

  it('clears the custom host when a configured host is selected and connects with it', async () => {
    const wrapper = await mountPage()

    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://custom:9200')
    await wrapper.find('[data-testid="connect-host"]').setValue('http://es:9200')

    expect((wrapper.find('[data-testid="connect-custom-host"]').element as HTMLInputElement).value).toBe('')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/connect', expect.objectContaining({
      body: { host: 'http://es:9200' },
    }))
  })

  it('sends trustCertificate on connect and navbar and navigates to overview', async () => {
    const wrapper = await mountPage()

    await wrapper.find('[data-testid="connect-trust-cert"]').setValue(true)
    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://es:9200')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/connect', expect.objectContaining({
      body: { host: 'https://es:9200', trustCertificate: true },
    }))
    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/navbar', expect.objectContaining({
      body: { host: 'https://es:9200', trustCertificate: true },
    }))
    expect((globalThis as any).navigateTo).toHaveBeenCalledWith('/overview')
  })

  it('shows a self-signed TLS hint when an untrusted connect fails with x509', async () => {
    const wrapper = await mountPage({
      '/connect': { status: 502, body: { error: 'x509: certificate signed by unknown authority' } },
    })

    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://es:9200')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const alert = wrapper.find('[data-testid="connect-error"]')
    expect(alert.exists()).toBe(true)
    expect(alert.attributes('data-color')).toBe('error')
    expect(alert.text()).toContain('self-signed certificate')
  })

  it('shows a generic error for other connect failures', async () => {
    const wrapper = await mountPage({
      '/connect': { status: 500, body: { error: 'boom' } },
    })

    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://es:9200')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const alert = wrapper.find('[data-testid="connect-error"]')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toBe('Connection failed')
  })

  it('sends credentials on both connect and navbar', async () => {
    const wrapper = await mountPage()

    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://es:9200')
    await wrapper.find('[data-testid="connect-username"]').setValue('admin')
    await wrapper.find('[data-testid="connect-password"]').setValue('secret')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/connect', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'admin', password: 'secret' },
    }))
    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/navbar', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'admin', password: 'secret' },
    }))
  })

  it('shows a credential rejection message on 401', async () => {
    const wrapper = await mountPage({
      '/connect': { status: 401, body: { error: 'unauthorized' } },
    })

    await wrapper.find('[data-testid="connect-custom-host"]').setValue('https://es:9200')
    await wrapper.find('[data-testid="connect-username"]').setValue('admin')
    await wrapper.find('[data-testid="connect-password"]').setValue('wrong')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    const alert = wrapper.find('[data-testid="connect-error"]')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toContain('cluster rejected credentials (401)')
  })
})
