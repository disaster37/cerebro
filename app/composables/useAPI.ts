export function useAPI() {
  const alerts = useState<any[]>('alerts', () => [])

  async function apiCall(method: string, url: string, body?: any) {
    try {
      const data = await $fetch(url, {
        method,
        body: body ? JSON.parse(JSON.stringify(body)) : undefined,
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
