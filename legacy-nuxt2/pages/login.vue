<template>
  <form class="card login-card" @submit.prevent="submit">
    <h1>ApplianceHub POS</h1>
    <p class="muted">Sign in to continue</p>
    <p v-if="checking" class="muted">Checking your session…</p>
    <template v-else>
      <div class="field"><label>Username</label><input v-model="username" autocomplete="username" autofocus></div>
      <div class="field"><label>Password</label><input v-model="password" type="password" autocomplete="current-password"></div>
      <p v-if="error" class="error">{{ error }}</p>
      <button class="btn" :disabled="busy || !username || !password">{{ busy ? 'Signing in…' : 'Sign in' }}</button>
    </template>
  </form>
</template>

<script>
import { safeRedirect, errMsg } from '~/utils/auth'

export default {
  layout: 'blank',
  data: () => ({ username: '', password: '', error: '', busy: false, checking: true }),
  async mounted() {
    // Silent sign-in: if only the short-lived access token expired, the refresh cookie restores the session.
    try {
      await this.$api.post('/api/auth/refresh')
      this.done()
    } catch (e) {
      this.checking = false
    }
  },
  methods: {
    done() {
      // Full page load on purpose: the gateway decides which app serves the target (it may be Vue 3).
      window.location.assign(safeRedirect(this.$route.query.redirect))
    },
    async submit() {
      this.busy = true
      this.error = ''
      try {
        await this.$api.post('/api/auth/login', { username: this.username, password: this.password })
        this.done()
      } catch (e) {
        this.error = errMsg(e)
        this.busy = false
      }
    },
  },
}
</script>
