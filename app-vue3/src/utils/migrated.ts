import manifest from '@/generated/migrated-routes.json'

const normalize = (p: string): string => p.split('?')[0].split('#')[0].replace(/\/+$/, '') || '/'

/** Mirrors the gateway rules generated from shared/migrated-routes.json */
export function isMigrated(path: string): boolean {
  const p = normalize(path)
  return manifest.routes.some((r) =>
    r.match === 'exact' ? p === r.path : p === r.path || p.startsWith(r.path + '/')
  )
}
