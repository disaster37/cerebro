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

  it('propagates credentials to navbar and cluster calls after clusterName changes', async () => {
    ;(globalThis as any).$fetch
      .mockResolvedValueOnce({ status: 200, body: {} })
      .mockResolvedValueOnce({ status: 200, body: { cluster_name: 'docker-cluster' } })
      .mockResolvedValueOnce({ status: 200, body: {} })

    const { connect, loadNavbar, clusterHost } = useCluster()
    await connect('https://es:9200', 'alice', 'pw', true)
    await loadNavbar('https://es:9200')

    expect((globalThis as any).$fetch).toHaveBeenNthCalledWith(2, '/navbar', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'alice', password: 'pw', trustCertificate: true },
    }))
    expect((globalThis as any).useState('clusterName').value).toBe('docker-cluster')
    expect(clusterHost.value).toBe('https://es:9200')

    await useAPI().apiCall('POST', '/overview', { host: clusterHost.value })
    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/overview', expect.objectContaining({
      body: { host: 'https://es:9200', username: 'alice', password: 'pw', trustCertificate: true },
    }))
  })

  it('failed connect to another host neither leaks nor mutates committed state', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { connect, connection, clusterName } = useCluster()
    await connect('https://es-a:9200', 'alice', 'pwA', true)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 401, body: { error: 'unauthorized' } })
    const ok = await connect('https://es-b:9200', 'bob', 'bad', false)

    expect(ok).toBe(false)
    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/connect', expect.objectContaining({
      body: { host: 'https://es-b:9200', username: 'bob', password: 'bad' },
    }))
    expect(connection.value).toEqual({
      host: 'https://es-a:9200',
      username: 'alice',
      password: 'pwA',
      trustCertificate: true,
    })
    expect(clusterName.value).toBe('https://es-a:9200')
    expect((globalThis as any).useState('trustCertificate').value).toBe(true)
  })

  it('successful connect without form credentials clears prior credentials', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { connect, connection } = useCluster()
    await connect('https://es:9200', 'alice', 'pw', true)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    await connect('prod')

    expect(connection.value).toEqual({
      host: 'prod',
      username: '',
      password: '',
      trustCertificate: false,
    })

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    await useAPI().apiCall('POST', '/overview', { host: 'prod' })
    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/overview', expect.objectContaining({
      body: { host: 'prod' },
    }))
  })

  it('does not inject credentials into calls for a different host', async () => {
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    const { connect } = useCluster()
    await connect('https://es:9200', 'alice', 'pw', true)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 200, body: {} })
    await useAPI().apiCall('POST', '/overview', { host: 'https://other:9200' })

    expect((globalThis as any).$fetch).toHaveBeenLastCalledWith('/overview', expect.objectContaining({
      body: { host: 'https://other:9200' },
    }))
  })

  it('pushes deduplicated danger alerts for 401/403/5xx and respects suppression', async () => {
    const { apiCall } = useAPI()
    const alerts = () => (globalThis as any).useState('alerts').value

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 401, body: { error: 'unauthorized' } })
    await apiCall('POST', '/overview', { host: 'x' })
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 401, body: { error: 'unauthorized' } })
    await apiCall('POST', '/overview', { host: 'x' })
    expect(alerts()).toHaveLength(1)
    expect(alerts()[0].type).toBe('danger')

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 403, body: { error: 'forbidden' } })
    await apiCall('POST', '/overview', { host: 'x' })
    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 502, body: { error: 'bad gateway' } })
    await apiCall('POST', '/overview', { host: 'x' })
    expect(alerts()).toHaveLength(3)
    expect(alerts()[2].message).toContain('bad gateway')

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 500, body: { error: 'boom' } })
    await apiCall('POST', '/overview', { host: 'x' }, { suppressErrorAlert: true })
    expect(alerts()).toHaveLength(3)

    ;(globalThis as any).$fetch.mockResolvedValueOnce({ status: 404, body: { error: 'not found' } })
    await apiCall('POST', '/overview', { host: 'x' })
    expect(alerts()).toHaveLength(3)
  })
})
