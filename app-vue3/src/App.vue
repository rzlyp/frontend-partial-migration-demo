<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import AppLink from '@/components/AppLink.vue'
import { MENU } from '@/utils/menu'
import { onLogout } from '@/utils/authSync'
import { useCartStore } from '@/stores/cart'
import { useSessionStore } from '@/stores/session'

const cart = useCartStore()
const session = useSessionStore()
const menu = computed(() => MENU.filter((m) => session.can(m.perm)))

let stop = () => {}
onMounted(() => {
  stop = onLogout(() => window.location.assign('/login'))
})
onBeforeUnmount(() => stop())
</script>

<template>
  <div v-if="session.ready" class="shell">
    <aside class="sidebar">
      <div class="brand">ApplianceHub <small>POS</small></div>
      <nav>
        <AppLink v-for="m in menu" :key="m.to" :to="m.to" class="nav-link">{{ m.label }}</AppLink>
      </nav>
      <div class="stack-badge stack-v3">Vue 3 · Vite (NEW)</div>
    </aside>
    <main class="content">
      <header class="topbar">
        <span>{{ session.outletName }}</span>
        <span class="userbox">
          <span class="cart-pill">Cart: {{ cart.count }} item(s)</span>
          <span>{{ session.user?.name }}</span>
          <span class="role">{{ session.user?.role }}</span>
          <button class="btn secondary" @click="session.logout()">Sign out</button>
        </span>
      </header>
      <div class="page"><RouterView /></div>
    </main>
  </div>
  <div v-else class="login-wrap">
    <p v-if="session.error" class="error">{{ session.error }}</p>
    <p v-else class="muted">Loading…</p>
  </div>
</template>
