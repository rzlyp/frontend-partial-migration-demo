import axios from 'axios'
import { broadcastLogout } from '~/utils/auth-sync'

// One refresh at a time per tab. The backend also tolerates concurrent refreshes (grace window).
let refreshing = null

const readCsrf = () => {
  const m = document.cookie.match(/(?:^|; )pos_csrf=([^;]+)/)
  return m ? decodeURIComponent(m[1]) : ''
}

export default ({ req, $config }, inject) => {
  const server = process.server
  const http = axios.create({
    // Browser: same origin through the gateway. SSR: straight to the API container.
    baseURL: server ? $config.apiInternalUrl : '',
    withCredentials: true,
    timeout: 10000,
  })

  http.interceptors.request.use((cfg) => {
    if (server) {
      // SSR acts on behalf of the user: forward their cookies (session JWT) to the API
      if (req && req.headers.cookie) cfg.headers.cookie = req.headers.cookie
    } else if (!/^(get|head|options)$/i.test(cfg.method || 'get')) {
      cfg.headers['X-CSRF-Token'] = readCsrf()
    }
    return cfg
  })

  http.interceptors.response.use(null, async (err) => {
    const cfg = err.config
    const status = err.response && err.response.status
    const isAuthCall = cfg && cfg.url && cfg.url.indexOf('/api/auth/') === 0
    if (!server && status === 401 && cfg && !cfg._retried && !isAuthCall) {
      cfg._retried = true
      try {
        refreshing = refreshing || http.post('/api/auth/refresh').finally(() => { refreshing = null })
        await refreshing
        return http(cfg) // replay the original request with the fresh cookie
      } catch (e) {
        window.location.assign('/login?redirect=' + encodeURIComponent(location.pathname + location.search))
      }
    }
    return Promise.reject(err)
  })

  const api = {
    get: (u, c) => http.get(u, c).then((r) => r.data),
    post: (u, d, c) => http.post(u, d, c).then((r) => r.data),
    put: (u, d, c) => http.put(u, d, c).then((r) => r.data),
    patch: (u, d, c) => http.patch(u, d, c).then((r) => r.data),
    delete: (u, c) => http.delete(u, c).then((r) => r.data),
    async logout() {
      try { await http.post('/api/auth/logout') } catch (e) { /* already signed out */ }
      broadcastLogout()
      window.location.assign('/login')
    },
  }
  inject('api', api) // -> context.$api, this.$api in components and store actions
}
