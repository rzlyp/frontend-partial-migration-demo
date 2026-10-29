// Route list owned by the Vue 3 repo and served by the gateway at GET /migrated-routes.json.
// Legacy never bundles it: moving a route must not require rebuilding this app.
const URL_PATH = '/migrated-routes.json'
const CACHE_KEY = 'pos.migrated.v1'
const SUPPORTED_VERSION = 1

let routes = null // null = not known yet (the navigation guard then uses its list-free fallback)

const normalize = (p) => p.split('?')[0].split('#')[0].replace(/\/+$/, '') || '/'

function accept(data) {
  if (!data || data.version !== SUPPORTED_VERSION || !Array.isArray(data.routes)) return false
  routes = data.routes.filter((r) => typeof r.path === 'string' && (r.match === 'exact' || r.match === 'prefix'))
  return true
}

/** Browser only. Use the last list we saw in this tab right away, then replace it with the fresh one. */
export async function loadMigratedRoutes() {
  try {
    const cached = sessionStorage.getItem(CACHE_KEY)
    if (cached) accept(JSON.parse(cached))
  } catch (e) { /* storage unavailable: carry on without it */ }
  try {
    const res = await fetch(URL_PATH, { cache: 'no-cache', credentials: 'same-origin' })
    if (!res.ok) return false
    const data = await res.json()
    if (!accept(data)) return false
    try { sessionStorage.setItem(CACHE_KEY, JSON.stringify(data)) } catch (e) { /* ignore */ }
    return true
  } catch (e) {
    return false // offline, direct access without the gateway, bad JSON: fallback rule still protects navigation
  }
}

/** Mirrors the gateway rules. Returns false while the list is unknown. */
export function isMigrated(path) {
  if (!routes) return false
  const p = normalize(path)
  return routes.some((r) => (r.match === 'exact' ? p === r.path : p === r.path || p.startsWith(r.path + '/')))
}
