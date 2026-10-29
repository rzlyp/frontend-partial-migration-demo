<template>
  <div>
    <h1>Products <small class="muted">(legacy)</small></h1>
    <div class="card">
      <table>
        <thead><tr><th>SKU</th><th>Name</th><th>Category</th><th>Price</th><th>Stock</th></tr></thead>
        <tbody>
          <tr v-for="p in products" :key="p.sku">
            <td>{{ p.sku }}</td><td>{{ p.name }}</td><td>{{ p.category }}</td><td>{{ fmt(p.price) }}</td><td>{{ p.stock }}</td>
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
    return { products: await $api.get('/api/products') }
  },
  methods: { fmt },
}
</script>
