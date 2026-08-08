import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useCluster } from '@/composables/useCluster'
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
})
