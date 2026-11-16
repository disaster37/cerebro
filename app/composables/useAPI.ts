export function useAPI() {
  const alerts = useState<any[]>('alerts', () => [])
  const trustCertificate = useState<boolean>('trustCertificate', () => false)

  async function apiCall(method: string, url: string, body?: any) {
    try {
      let payload: any = undefined
      if (body !== undefined) {
        payload = JSON.parse(JSON.stringify(body))
        if (
          trustCertificate.value &&
          payload !== null &&
          typeof payload === 'object' &&
          !Array.isArray(payload) &&
          payload.trustCertificate === undefined
        ) {
          payload.trustCertificate = true
        }
      }

      const data = await $fetch(url, {
        method,
        body: payload,
        headers: { 'Accept': 'application/json', 'Content-Type': 'application/json' },
      }) as any
      if (data.status === 303) {
        navigateTo('/login')
        return null
      }
      return data
    } catch (err: any) {
      alerts.value.push({ message: err.message || 'Request failed', type: 'danger' })
      return null
    }
  }

  return { apiCall }
}
