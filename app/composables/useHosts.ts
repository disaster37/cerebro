export function useHosts() {
  const { apiCall } = useAPI()
  const hosts = ref<string[]>([])

  async function load() {
    const resp = await apiCall('GET', '/connect/hosts')
    if (resp?.status === 200) {
      hosts.value = resp.body || []
    }
  }

  return { hosts, load }
}
