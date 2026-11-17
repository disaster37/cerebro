import { extractDetailedError } from '../utils/format'

export interface ClusterConnection {
  host: string
  username: string
  password: string
  trustCertificate: boolean
}

export interface ApiCallOptions {
  suppressErrorAlert?: boolean
}

export function defaultClusterConnection(): ClusterConnection {
  return {
    host: '',
    username: '',
    password: '',
    trustCertificate: false,
  }
}

export function extractEnvelopeError(body: any): string {
  return extractDetailedError(body)
}

function pushUniqueAlert(alerts: any[], message: string, type: string) {
  if (!alerts.some((a) => a.message === message && a.type === type)) {
    alerts.push({ message, type })
  }
}

export function useAPI() {
  const alerts = useState<any[]>('alerts', () => [])
  const connection = useState<ClusterConnection>('clusterConnection', defaultClusterConnection)

  async function apiCall(method: string, url: string, body?: any, options?: ApiCallOptions) {
    try {
      let payload: any = undefined
      if (body !== undefined) {
        payload = JSON.parse(JSON.stringify(body))
        const conn = connection.value
        if (
          url !== '/connect' &&
          payload !== null &&
          typeof payload === 'object' &&
          !Array.isArray(payload) &&
          conn?.host &&
          payload.host === conn.host
        ) {
          if (conn.username && conn.password) {
            if (payload.username === undefined) payload.username = conn.username
            if (payload.password === undefined) payload.password = conn.password
          }
          if (conn.trustCertificate && payload.trustCertificate === undefined) {
            payload.trustCertificate = true
          }
        }
      }

      const data = await $fetch(url, {
        method,
        body: payload,
        headers: { 'Accept': 'application/json', 'Content-Type': 'application/json' },
      }) as any
      if (data?.status === 303) {
        navigateTo('/login')
        return null
      }

      if (!options?.suppressErrorAlert && data && typeof data.status === 'number') {
        const detail = extractEnvelopeError(data.body)
        if (data.status === 401) {
          pushUniqueAlert(
            alerts.value,
            'Cluster authentication failed (401 Unauthorized). Reconnect from the Connect page with valid credentials.',
            'danger',
          )
        } else if (data.status === 403) {
          pushUniqueAlert(
            alerts.value,
            'Cluster request forbidden (403 Forbidden). Verify that the connected user has sufficient cluster permissions.',
            'danger',
          )
        } else if (data.status >= 500) {
          pushUniqueAlert(
            alerts.value,
            detail ? `Cluster request failed (${data.status}): ${detail}` : `Cluster request failed (${data.status}).`,
            'danger',
          )
        }
      }

      return data
    } catch (err: any) {
      pushUniqueAlert(alerts.value, err.message || 'Request failed', 'danger')
      return null
    }
  }

  return { apiCall }
}
