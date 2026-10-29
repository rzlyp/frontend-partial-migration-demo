import { createRouter, createWebHistory } from 'vue-router'
import { isMigrated } from '@/utils/migrated'
import { useCartStore } from '@/stores/cart'
import { useSessionStore } from '@/stores/session'
import SettingsPage from '@/pages/SettingsPage.vue'
import ProtectionPlanPage from '@/pages/ProtectionPlanPage.vue'
import LegacyFallback from '@/pages/LegacyFallback.vue'

export const router = createRouter({
  // Router base is "/" (page URLs must be identical to the legacy app's.
  history: createWebHistory('/'),
  routes: [
    { path: '/settings', component: SettingsPage },
    { path: '/settings/protection-plan', component: ProtectionPlanPage },
    { path: '/:pathMatch(.*)*', component: LegacyFallback },
  ],
})

router.beforeEach(async (to, from) => {
  // Mirror of the Nuxt 2 plugin: an in-app navigation to a route Vue 3 does not own is a full page load.
  // Skipped on the first navigation (from.matched is empty) so a misconfigured gateway can't cause a reload loop.
  if (from.matched.length > 0 && !isMigrated(to.path)) {
    window.location.assign(to.fullPath)
    return false
  }
  // Auth: the cookie is the credential, /api/me the source of truth. A 401 (after a silent refresh attempt)
  // has already sent the browser to /login?redirect=..., so there is nothing more to do here.
  try {
    await useSessionStore().init()
    await useCartStore().ensureLoaded()
  } catch {
    return false
  }
})
