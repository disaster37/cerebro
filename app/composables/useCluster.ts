import { ref } from 'vue'
import { useAPI } from './useAPI'

export function useCluster() {
  const clusterName = useState<string>('clusterName', () => '')
  const trustCertificate = useState<boolean>('trustCertificate', () => false)
  const lastConnectError = ref<string>('')
  const { apiCall } = useAPI()

  async function connect(
    host: string,
    username?: string,
    password?: string,
    trustCert?: boolean,
  ): Promise<boolean> {
    lastConnectError.value = ''
    const prevTrust = trustCertificate.value
    const nextTrust = Boolean(trustCert)
    trustCertificate.value = nextTrust

    const body: Record<string, any> = { host }
    if (username) body.username = username
    if (password) body.password = password
    if (nextTrust) body.trustCertificate = true

    const resp = await apiCall('POST', '/connect', body)
    if (resp?.status === 200) {
      clusterName.value = host
      return true
    }

    trustCertificate.value = prevTrust
    lastConnectError.value = typeof resp?.body?.error === 'string' ? resp.body.error : ''
    return false
  }

  async function loadNavbar(host: string) {
    const resp = await apiCall('POST', '/navbar', { host })
    if (resp?.status === 200) {
      clusterName.value = resp.body?.cluster_name || host
      if (resp.body?.username) {
        useState<string>('user').value = resp.body.username
      }
      useState<string>('backendType').value = resp.body?.backendType || ''
      useState<string>('backendVersion').value = resp.body?.backendVersion || ''
    }
  }

  return { clusterName, trustCertificate, lastConnectError, connect, loadNavbar }
}
