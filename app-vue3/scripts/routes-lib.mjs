// Shared by app-vue3 (prebuild) and the repo-level generator (gateway outputs).
import { readFileSync } from 'node:fs'

export function loadRoutes(file) {
  const manifest = JSON.parse(readFileSync(file, 'utf8'))
  const seen = new Set()
  for (const r of manifest.routes) {
    if (!r.path?.startsWith('/') || r.path === '/' || r.path.endsWith('/'))
      throw new Error(`Invalid path "${r.path}" (must start with "/", not be "/" and have no trailing slash)`)
    if (!['exact', 'prefix'].includes(r.match)) throw new Error(`Invalid match "${r.match}" for ${r.path}`)
    if (seen.has(r.path)) throw new Error(`Duplicate path ${r.path}`)
    seen.add(r.path)
  }
  return manifest.routes
}

const slim = (routes) => routes.map(({ path, match }) => ({ path, match }))

/** The slim form bundled into the Vue 3 app (no notes). */
export const toAppManifest = (routes) => JSON.stringify({ routes: slim(routes) }, null, 2) + '\n'

/** Contract with legacy: GET /migrated-routes.json on the gateway. Additive changes only; bump version for breaking ones. */
export const MANIFEST_VERSION = 1
export const toGatewayManifest = (routes) =>
  JSON.stringify({ version: MANIFEST_VERSION, routes: slim(routes) }, null, 2) + '\n'
