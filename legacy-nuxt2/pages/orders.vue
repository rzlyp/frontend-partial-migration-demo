<template>
  <div>
    <h1>Orders <small class="muted">(legacy)</small></h1>
    <div class="card">
      <table>
        <thead><tr><th>Order</th><th>Time</th><th>Cashier</th><th>Items</th><th>Payment</th><th>Total</th></tr></thead>
        <tbody>
          <tr v-for="o in orders" :key="o.id">
            <td>{{ o.id }}</td>
            <td>{{ new Date(o.createdAt).toLocaleString('id-ID') }}</td>
            <td>{{ o.cashierName }}</td>
            <td>{{ o.items.reduce((n, i) => n + i.qty, 0) }}</td>
            <td>{{ o.paymentMethod }}</td>
            <td>{{ fmt(o.total) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
import { fmt } from '~/utils/money'

export default {
  async asyncData({ $api }) {
    return { orders: await $api.get('/api/orders') }
  },
  methods: { fmt },
}
</script>
