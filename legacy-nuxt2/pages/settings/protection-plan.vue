<template>
  <div>
    <h1>Protection Plans <small class="muted">(LEGACY version, rollback target)</small></h1>
    <div class="card">
      <p v-if="!items.length" class="muted">Cart is empty. Add appliances in POS first.</p>
      <table v-else>
        <tbody>
          <tr v-for="i in items" :key="i.sku">
            <td><b>{{ i.name }}</b><div class="muted">{{ fmt(i.price) }} × {{ i.qty }}</div></td>
            <td>
              <div class="plan-options">
                <label><input type="radio" :name="i.sku" :checked="!i.plan" @change="choose(i, '')"> No plan</label>
                <label v-for="p in i.planOptions" :key="p.id">
                  <input type="radio" :name="i.sku" :checked="i.plan && i.plan.id === p.id" @change="choose(i, p.id)">
                  {{ p.name }} — {{ fmt(p.price) }}
                </label>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
      <div class="total"><span>Cart total</span><span>{{ fmt(total) }}</span></div>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script>
import { mapState } from 'vuex'
import { fmt } from '~/utils/money'
import { errMsg } from '~/utils/auth'

export default {
  data: () => ({ error: '' }),
  computed: { ...mapState('cart', ['items', 'total']) },
  methods: {
    fmt,
    async choose(item, planId) {
      this.error = ''
      try { await this.$store.dispatch('cart/setPlan', { sku: item.sku, planId }) } catch (e) { this.error = errMsg(e) }
    },
  },
}
</script>
