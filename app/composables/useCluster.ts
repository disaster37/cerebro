import { computed, ref } from 'vue'
import { defaultClusterConnection, useAPI, type ClusterConnection } from './useAPI'

export function useCluster() {
  const connection = useState<ClusterConnection>('clusterConnection', defaultClusterConnection)
  const clusterName = useState<string>('clusterName', () => '')
  const trustCertificate = useState<boolean>('trustCertificate', () => false)
  const backendType = useState<string>('backendType', () => '')
  const backendVersion = useState<string>('backendVersion', () => '')
  const alerts = useState<any[]>('alerts', () => [])
  const lastConnectError = ref<string>('')
  const lastConnectStatus = ref<number>(0)
  const { apiCall } = useAPI()

  const clusterHost = computed(() => connection.value.host || clusterName.value)

  async function connect(
    host: string,
    username?: string,
    password?: string,
    trustCert?: boolean,
  ): Promise<boolean> {
    lastConnectError.value = ''
    lastConnectStatus.value = 0

    const nextTrust = Boolean(trustCert)
    const hasCreds = Boolean(username && password)
    const body: Record<string, any> = { host }
    if (username) body.username = username
    if (password) body.password = password
    if (nextTrust) body.trustCertificate = true

    const resp = await apiCall('POST', '/connect', body, { suppressErrorAlert: true })
    if (resp?.status === 200) {
      connection.value = {
        host,
        username: hasCreds ? username! : '',
        password: hasCreds ? password! : '',
        trustCertificate: nextTrust,
      }
      trustCertificate.value = nextTrust
      clusterName.value = host
      backendType.value = ''
      backendVersion.value = ''
      alerts.value = []
      return true
    }

    lastConnectStatus.value = typeof resp?.status === 'number' ? resp.status : 0
    lastConnectError.value = typeof resp?.body?.error === 'string' ? resp.body.error : ''
    return false
  }

  async function loadNavbar(host: string) {
    const resp = await apiCall('POST', '/navbar', { host })
    if (resp?.status === 200) {
      clusterName.value = resp.body?.cluster_name || host
      if (resp.body?.username) {
        useState<string>('user', () => '').value = resp.body.username
      }
      backendType.value = resp.body?.backendType || ''
      backendVersion.value = resp.body?.backendVersion || ''
    }
  }

  function clearConnection() {
    connection.value = defaultClusterConnection()
    trustCertificate.value = false
    clusterName.value = ''
    backendType.value = ''
    backendVersion.value = ''
  }

  return {
    connection,
    clusterHost,
    clusterName,
    trustCertificate,
    lastConnectError,
    lastConnectStatus,
    connect,
    loadNavbar,
    clearConnection,
  }
}
