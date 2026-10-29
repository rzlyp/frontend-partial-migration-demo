// Runs on every route (server during SSR, client on SPA navigation).
export default async function ({ store, $api, route, redirect }) {
  if (route.path === '/login') return
  if (store.state.session.user) return // already loaded (state hydrated from SSR / earlier navigation)
  try {
    const [me, cart] = await Promise.all([$api.get('/api/me'), $api.get('/api/cart')])
    store.commit('session/setMe', me)
    store.commit('cart/setCart', cart)
  } catch (e) {
    const status = e.response && e.response.status
    if (status === 401 || status === 403) return redirect('/login', { redirect: route.fullPath })
    throw e
  }
}
