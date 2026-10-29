import { isMigrated, loadMigratedRoutes } from '~/utils/migrated'

// Strangler-fig glue: a client-side navigation INTO a route owned by Vue 3 must be a real page load,
// so the gateway can hand the request to the other app.
export default ({ app }) => {
  loadMigratedRoutes() // async, fire and forget: the list comes from the gateway, not from this bundle

  app.router.beforeEach((to, from, next) => {
    // from.matched is empty only on the very first navigation. Never redirect then, otherwise a
    // gateway/list mismatch would cause an infinite reload loop.
    const inApp = from.matched.length > 0
    // 1) the list says Vue 3 owns it; 2) fallback needing no list: legacy has no such route at all
    //    (it was removed from here), so only the gateway can know who serves it.
    if (inApp && (isMigrated(to.path) || to.matched.length === 0)) {
      window.location.assign(to.fullPath)
      return // intentionally never call next(): the page is about to unload
    }
    next()
  })
}
