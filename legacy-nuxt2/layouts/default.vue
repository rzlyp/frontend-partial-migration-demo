<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">ApplianceHub <small>POS</small></div>
      <nav>
        <nuxt-link v-for="m in menu" :key="m.to" :to="m.to" exact class="nav-link">{{ m.label }}</nuxt-link>
      </nav>
      <div class="stack-badge stack-legacy">Nuxt 2 · Vue 2 (LEGACY)</div>
    </aside>
    <main class="content">
      <header class="topbar">
        <span>{{ storeName }}</span>
        <span class="userbox">
          <span class="cart-pill">Cart: {{ count }} item(s)</span>
          <span>{{ user && user.name }}</span>
          <span class="role">{{ user && user.role }}</span>
          <button class="btn secondary" @click="$api.logout()">Sign out</button>
        </span>
      </header>
      <div class="page"><nuxt /></div>
    </main>
  </div>
</template>

<script>
import { MENU } from '~/utils/menu'
import { onLogout } from '~/utils/auth-sync'

export default {
  data: () => ({ stop: null }),
  computed: {
    user() { return this.$store.state.session.user },
    storeName() { return this.$store.state.session.outlet.name },
    count() { return this.$store.state.cart.count },
    menu() { return MENU.filter((m) => this.$store.getters['session/can'](m.perm)) },
  },
  mounted() { this.stop = onLogout(() => window.location.assign('/login')) },
  beforeDestroy() { if (this.stop) this.stop() },
}
</script>
