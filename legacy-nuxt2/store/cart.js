// The cart lives on the server (single source of truth, shared by legacy and Vue 3). The store only mirrors it.
export const state = () => ({ items: [], count: 0, total: 0 })

export const mutations = {
  setCart(s, cart) { s.items = cart.items; s.count = cart.count; s.total = cart.total },
}

export const actions = {
  async refresh({ commit }) { commit('setCart', await this.$api.get('/api/cart')) },
  async add({ commit }, sku) { commit('setCart', await this.$api.post('/api/cart/items', { sku })) },
  async setQty({ commit }, { sku, qty }) { commit('setCart', await this.$api.patch('/api/cart/items/' + sku, { qty })) },
  async setPlan({ commit }, { sku, planId }) {
    commit('setCart', await this.$api.put('/api/cart/items/' + sku + '/plan', { planId: planId || '' }))
  },
  async clear({ commit }) { commit('setCart', await this.$api.delete('/api/cart')) },
}
