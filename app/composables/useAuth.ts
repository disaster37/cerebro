import { ref } from 'vue'
import { defaultClusterConnection, type ClusterConnection } from './useAPI'

export function sanitizeRedirectPath(raw?: string | null): string {
  if (!raw) return '/'
  let decoded = raw
  try {
    decoded = decodeURIComponent(raw)
  } catch {
    return '/'
  }
  if (
    !decoded.startsWith('/') ||
    decoded.startsWith('//') ||
    decoded.startsWith('/\\')
  ) {
    return '/'
  }
  for (let i = 0; i < decoded.length; i++) {
    const code = decoded.charCodeAt(i)
    if (code < 0x20 || code === 0x7f) return '/'
  }
  return decoded
}

export function consumeRedirectCookie(): string {
  if (typeof document === 'undefined' || !document.cookie) return '/'
  const match = document.cookie
    .split(';')
    .map((part) => part.trim())
    .find((part) => part.startsWith('redirect='))
  if (!match) return '/'
  const value = match.slice('redirect='.length)
  document.cookie = 'redirect=; Max-Age=0; path=/; SameSite=Lax'
  return sanitizeRedirectPath(value)
}

export function useAuth() {
  const user = useState<string>('user', () => '')
  const authEnabled = useState<boolean>('authEnabled', () => false)
  const sessionChecked = ref(false)

  async function checkSession(): Promise<string> {
    try {
      const resp = await $fetch('/auth/me', {
        method: 'GET',
        headers: { Accept: 'application/json' },
      }) as any
      if (resp?.status === 200 && resp.body && typeof resp.body === 'object') {
        authEnabled.value = Boolean(resp.body.authEnabled)
        user.value = typeof resp.body.username === 'string' ? resp.body.username : ''
        sessionChecked.value = true
        return user.value
      }
    } catch {
      // Ignore transient bootstrap failures
    }
    return user.value
  }

  async function login(username: string, password: string): Promise<boolean> {
    const form = new FormData()
    form.append('user', username)
    form.append('password', password)
    try {
      await fetch('/auth/login', {
        method: 'POST',
        body: form,
        redirect: 'manual',
        credentials: 'same-origin',
      })
    } catch {
      return false
    }
    const sessionUser = await checkSession()
    // Only treat the login as a no-op success when /auth/me positively
    // confirmed that Cerebro auth is disabled. A failed checkSession() leaves
    // authEnabled at its previous value (false on a fresh load), so relying on
    // !authEnabled alone would fail open and report a successful login without
    // an established session.
    if (sessionChecked.value && !authEnabled.value) {
      return true
    }
    return Boolean(sessionUser)
  }

  async function logout(): Promise<void> {
    try {
      await $fetch('/auth/logout', { method: 'POST' })
    } catch {
      // /auth/logout responds 302 -> /login HTML; ignore non-JSON parse errors
    }
    user.value = ''
    useState<ClusterConnection>('clusterConnection', defaultClusterConnection).value = defaultClusterConnection()
    useState<boolean>('trustCertificate', () => false).value = false
    useState<string>('clusterName', () => '').value = ''
    useState<string>('backendType', () => '').value = ''
    useState<string>('backendVersion', () => '').value = ''
    await navigateTo('/login')
  }

  return { user, authEnabled, login, logout, checkSession }
}
