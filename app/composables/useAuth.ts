export function useAuth() {
  const user = useState<string>('user', () => '')

  async function login(username: string, password: string) {
    const form = new FormData()
    form.append('user', username)
    form.append('password', password)
    const resp = await fetch('/auth/login', { method: 'POST', body: form, redirect: 'manual' })
    if (resp.status === 302 || resp.redirected) {
      user.value = username
      return true
    }
    return false
  }

  async function checkSession() {
    try {
      const resp = await $fetch('/navbar', {
        method: 'POST',
        body: { host: '' },
      }) as any
      if (resp?.body?.username) {
        user.value = resp.body.username
      }
    } catch { /* ignore */ }
  }

  return { user, login, checkSession }
}
