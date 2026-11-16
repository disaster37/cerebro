import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useCluster } from '@/composables/useCluster'
import { useAPI } from '@/composables/useAPI'
import { resetTestStubs } from '../setup'

describe('useCluster', () => {
  beforeEach(() => {
    resetTestStubs()
    ;(globalThis as any).$fetch.mockReset()
  })

  it('loadNavbar sets clusterName, user, backendType, backendVersion', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 200,
      body: {
        cluster_name: 'docker-cluster',
        username: 'u',
        backendType: 'opensearch',
        backendVersion: '3.4.0',
      },
    })

    const { loadNavbar } = useCluster()
    await loadNavbar('http://es:9200')

    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/navbar', expect.objectContaining({
      method: 'POST',
      body: { host: 'http://es:9200' },
    }))
    const get = (k: string) => (globalThis as any).useState(k).value
    expect(get('clusterName')).toBe('docker-cluster')
    expect(get('user')).toBe('u')
    expect(get('backendType')).toBe('opensearch')
    expect(get('backendVersion')).toBe('3.4.0')
  })

  it('connect returns true and sets clusterName on success', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { connect } = useCluster()
    const ok = await connect('http://es:9200')
    expect(ok).toBe(true)
    expect((globalThis as any).useState('clusterName').value).toBe('http://es:9200')
  })

  it('connect returns false on failure', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 502, body: { error: 'x' } })
    const { connect } = useCluster()
    const ok = await connect('http://es:9200')
    expect(ok).toBe(false)
  })

  it('connect sends trustCertificate and sets state when opted in', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { connect } = useCluster()
    const ok = await connect('https://es:9200', 'admin', 'secret', true)

    expect(ok).toBe(true)
    expect((globalThis as any).$fetch).toHaveBeenCalledWith('/connect', expect.objectContaining({
      method: 'POST',
      body: {
        host: 'https://es:9200',
        username: 'admin',
        password: 'secret',
        trustCertificate: true,
      },
    }))
    expect((globalThis as any).useState('trustCertificate').value).toBe(true)
  })

  it('loadNavbar propagates trustCertificate after a trusted connect', async () => {
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: {} })
      .mockResolvedValueOnce({ status: 200, body: {} })

    const { connect, loadNavbar } = useCluster()
    await connect('https://es:9200', undefined, undefined, true)
    await loadNavbar('https://es:9200')

    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/navbar', expect.objectContaining({
      method: 'POST',
      body: { host: 'https://es:9200', trustCertificate: true },
    }))
  })

  it('apiCall leaves bodyless GET requests untouched when trustCertificate is true', async () => {
    ;(globalThis as any).useState('trustCertificate').value = true
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: [] })

    const { apiCall } = useAPI()
    await apiCall('GET', '/connect/hosts')

    const call = (globalThis as any).$fetch.mock.calls[0]
    expect(call[0]).toBe('/connect/hosts')
    expect(call[1].body).toBeUndefined()
  })

  it('switching to an untrusted cluster rolls back trust on failure and resets it on success', async () => {
    const { connect, loadNavbar, lastConnectError } = useCluster()

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    await connect('https://es:9200', undefined, undefined, true)
    expect((globalThis as any).useState('trustCertificate').value).toBe(true)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({
      status: 502,
      body: { error: 'x509: certificate signed by unknown authority' },
    })
    const failed = await connect('https://es2:9200', undefined, undefined, false)

    expect(failed).toBe(false)
    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/connect', expect.objectContaining({
      body: { host: 'https://es2:9200' },
    }))
    expect((globalThis as any).useState('trustCertificate').value).toBe(true)
    expect(lastConnectError.value).toBe('x509: certificate signed by unknown authority')

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const succeeded = await connect('https://es2:9200', undefined, undefined, false)

    expect(succeeded).toBe(true)
    expect((globalThis as any).useState('trustCertificate').value).toBe(false)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    await loadNavbar('https://es2:9200')
    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/navbar', expect.objectContaining({
      body: { host: 'https://es2:9200' },
    }))
  })
})
