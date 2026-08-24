import { useAPI } from './useAPI'

export function useCluster() {
  const clusterName = useState<string>('clusterName', () => '')
  const { apiCall } = useAPI()

  async function connect(host: string, username?: string, password?: string) {
    const body: any = { host }
    if (username) body.username = username
    if (password) body.password = password
    const resp = await apiCall('POST', '/connect', body)
    if (resp?.status === 200) {
      clusterName.value = host
      return true
    }
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

  return { clusterName, connect, loadNavbar }
}
