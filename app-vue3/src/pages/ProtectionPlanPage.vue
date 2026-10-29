<script setup lang="ts">
import { ref } from 'vue'
import AppLink from '@/components/AppLink.vue'
import { errMsg } from '@/api/http'
import { useCartStore } from '@/stores/cart'
import { fmt } from '@/utils/money'

const cart = useCartStore()
const error = ref('')

async function choose(sku: string, planId: string | null) {
  error.value = ''
  try {
    await cart.setPlan(sku, planId)
  } catch (e) {
    error.value = errMsg(e)
  }
}
</script>

<template>
  <h1>Protection Plans <small class="muted">(Vue 3)</small></h1>
  <div class="card">
    <p v-if="!cart.items.length" class="muted">
      Cart is empty. Add appliances in <AppLink to="/pos" class="link">POS</AppLink> first.
    </p>
    <table v-else>
      <tbody>
        <tr v-for="item in cart.items" :key="item.sku">
          <td>
            <b>{{ item.name }}</b>
            <div class="muted">{{ fmt(item.price) }} × {{ item.qty }}</div>
          </td>
          <td>
            <div class="plan-options">
              <label>
                <input type="radio" :name="item.sku" :checked="!item.plan" @change="choose(item.sku, null)" />
                No plan
              </label>
              <label v-for="p in item.planOptions" :key="p.id">
                <input type="radio" :name="item.sku" :checked="item.plan?.id === p.id" @change="choose(item.sku, p.id)" />
                {{ p.name }} — {{ fmt(p.price) }}
              </label>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
    <div class="total"><span>Cart total</span><span>{{ fmt(cart.total) }}</span></div>
    <p v-if="error" class="error">{{ error }}</p>
    <p><AppLink to="/pos" class="link">← Back to POS (legacy, full page load)</AppLink></p>
  </div>
</template>
