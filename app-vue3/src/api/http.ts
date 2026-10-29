// Same-origin API client. The session lives in HttpOnly cookies, so JS never touches a token.
export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

// One refresh at a time per tab. The backend also tolerates concurrent refreshes (grace window).
let refreshing: Promise<void> | null = null

const csrfToken = (): string => {
  const m = document.cookie.match(/(?:^|; )pos_csrf=([^;]+)/)
  return m ? decodeURIComponent(m[1]) : ''
}

export function goToLogin(): void {
  // Login is still owned by the legacy app: a full page load lets the gateway route it.
  window.location.assign('/login?redirect=' + encodeURIComponent(location.pathname + location.search))
}

function send(method: string, url: string, body?: unknown): Promise<Response> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (method !== 'GET' && method !== 'HEAD') headers['X-CSRF-Token'] = csrfToken()
  return fetch(url, {
    method,
    headers,
    credentials: 'same-origin',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

function refreshSession(): Promise<void> {
  refreshing ??= send('POST', '/api/auth/refresh')
    .then((r) => {
      if (!r.ok) throw new ApiError(r.status, 'refresh failed')
    })
    .finally(() => {
      refreshing = null
    })
  return refreshing
}

export async function request<T>(method: string, url: string, body?: unknown): Promise<T> {
  let res = await send(method, url, body)
  if (res.status === 401 && !url.startsWith('/api/auth/')) {
    try {
      await refreshSession()
      res = await send(method, url, body) // replay once with the fresh cookie
    } catch {
      goToLogin()
      throw new ApiError(401, 'session expired')
    }
  }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => null)
  if (!res.ok) throw new ApiError(res.status, data?.error ?? res.statusText)
  return data as T
}

export const api = {
  get: <T>(url: string) => request<T>('GET', url),
  post: <T>(url: string, body?: unknown) => request<T>('POST', url, body),
  put: <T>(url: string, body?: unknown) => request<T>('PUT', url, body),
  patch: <T>(url: string, body?: unknown) => request<T>('PATCH', url, body),
  del: <T>(url: string) => request<T>('DELETE', url),
}

export const errMsg = (e: unknown): string => (e instanceof Error ? e.message : 'Something went wrong')
