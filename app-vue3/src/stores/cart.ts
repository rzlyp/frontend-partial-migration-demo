import { defineStore } from 'pinia'
import { api } from '@/api/http'

// The cart lives on the server (single source of truth shared with the legacy app). This store mirrors it.
export interface PlanView { id: string; name: string; price: number }
export interface CartItem {
  sku: string; name: string; price: number; qty: number
  plan: PlanView | null
  planOptions: PlanView[]
  lineTotal: number
}
interface Cart { items: CartItem[]; count: number; total: number }

export const useCartStore = defineStore('cart', {
  state: (): Cart & { loaded: boolean } => ({ items: [], count: 0, total: 0, loaded: false }),
  actions: {
    apply(cart: Cart) {
      this.items = cart.items
      this.count = cart.count
      this.total = cart.total
      this.loaded = true
    },
    async ensureLoaded() {
      if (!this.loaded) this.apply(await api.get<Cart>('/api/cart'))
    },
    async setPlan(sku: string, planId: string | null) {
      this.apply(await api.put<Cart>(`/api/cart/items/${sku}/plan`, { planId: planId ?? '' }))
    },
  },
})
