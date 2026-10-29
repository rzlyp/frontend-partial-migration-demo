<template>
  <div>
    <h1>Dashboard <small class="muted">(legacy)</small></h1>
    <div class="kpis">
      <div class="card kpi"><span class="muted">Today's sales</span><b>{{ fmt(stats.todaySales) }}</b></div>
      <div class="card kpi"><span class="muted">Orders today</span><b>{{ stats.ordersToday }}</b></div>
      <div class="card kpi"><span class="muted">Protection plans sold</span><b>{{ stats.plansSold }}</b></div>
      <div class="card kpi"><span class="muted">Low stock items</span><b>{{ stats.lowStock }}</b></div>
    </div>
    <div class="card">
      <h2>Migration demo</h2>
      <p class="muted">
        <b>Settings</b> and <b>Protection Plans</b> are served by Vue 3. Same URL, same session cookie, same API.
        This dashboard was rendered on the server (SSR) using your cookie, forwarded to the API.
      </p>
    </div>
  </div>
</template>

<script>
import { fmt } from '~/utils/money'

export default {
  async asyncData({ $api }) {
    return { stats: await $api.get('/api/dashboard') }
  },
  methods: { fmt },
}
</script>
