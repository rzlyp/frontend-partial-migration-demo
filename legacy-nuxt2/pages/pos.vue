<template>
  <div>
    <h1>POS / Checkout <small class="muted">(legacy)</small></h1>
    <div class="pos-grid">
      <section class="card">
        <h2>Appliances</h2>
        <div class="products">
          <button v-for="p in products" :key="p.sku" class="product" :disabled="p.stock < 1" @click="run(() => add(p.sku))">
            <strong>{{ p.name }}</strong>
            <span class="muted">{{ p.category }} · {{ p.sku }} · stock {{ p.stock }}</span>
            <span>{{ fmt(p.price) }}</span>
          </button>
        </div>
      </section>

      <section class="card">
        <h2>Cart</h2>
        <p v-if="!items.length" class="muted">Cart is empty. Tap an appliance to add it.</p>
        <table v-else>
          <tbody>
            <tr v-for="i in items" :key="i.sku">
              <td>
                {{ i.name }}
                <div v-if="i.plan" class="muted">+ {{ i.plan.name }} ({{ fmt(i.plan.price) }})</div>
              </td>
              <td>
                <button class="btn secondary" @click="run(() => setQty({ sku: i.sku, qty: i.qty - 1 }))">−</button>
                {{ i.qty }}
                <button class="btn secondary" @click="run(() => setQty({ sku: i.sku, qty: i.qty + 1 }))">+</button>
              </td>
              <td>{{ fmt(i.lineTotal) }}</td>
            </tr>
          </tbody>
        </table>
        <div class="total"><span>Total</span><span>{{ fmt(total) }}</span></div>
        <p>
          <!-- A plain nuxt-link: the global guard turns it into a full page load into Vue 3 -->
          <nuxt-link class="link" to="/settings/protection-plan">Add protection plans →</nuxt-link>
        </p>
        <div class="actions">
          <select v-model="method">
            <option value="cash">Cash</option><option value="card">Card</option><option value="qris">QRIS</option>
          </select>
          <button class="btn" :disabled="!items.length" @click="checkout">Pay {{ fmt(total) }}</button>
          <button class="btn danger" :disabled="!items.length" @click="run(clear)">Clear</button>
        </div>
        <p v-if="error" class="error">{{ error }}</p>
      </section>
    </div>

    <section v-if="receipt" class="card receipt">
      <h2>Receipt {{ receipt.id }} <span class="ok">paid by {{ receipt.paymentMethod }}</span></h2>
      <div v-for="i in receipt.items" :key="i.sku" class="row">
        <span>{{ i.qty }} × {{ i.name }}<template v-if="i.plan"> + {{ i.plan.name }}</template></span><span>{{ fmt(i.lineTotal) }}</span>
      </div>
      <div class="row"><b>Total</b><b>{{ fmt(receipt.total) }}</b></div>
      <p class="muted">Cashier: {{ receipt.cashierName }} — {{ receipt.receiptFooter }}</p>
    </section>
  </div>
</template>

<script>
import { mapActions, mapState } from 'vuex'
import { fmt } from '~/utils/money'
import { errMsg } from '~/utils/auth'

export default {
  async asyncData({ $api }) {
    return { products: await $api.get('/api/products') }
  },
  data: () => ({ method: 'cash', error: '', receipt: null }),
  computed: { ...mapState('cart', ['items', 'total']) },
  methods: {
    fmt,
    ...mapActions('cart', ['add', 'setQty', 'clear', 'refresh']),
    async run(fn) {
      this.error = ''
      try { await fn() } catch (e) { this.error = errMsg(e) }
    },
    async checkout() {
      await this.run(async () => {
        this.receipt = await this.$api.post('/api/orders', { paymentMethod: this.method })
        await this.refresh()
        this.products = await this.$api.get('/api/products') // stock changed
      })
    },
  },
}
</script>
