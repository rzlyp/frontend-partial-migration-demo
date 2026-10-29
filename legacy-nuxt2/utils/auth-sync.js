// Cross-tab logout (same origin, so it also reaches the Vue 3 app in another tab).
const NAME = 'pos-auth'

export function broadcastLogout() {
  try { const c = new BroadcastChannel(NAME); c.postMessage('logout'); c.close() } catch (e) { /* unsupported */ }
}

export function onLogout(cb) {
  try {
    const c = new BroadcastChannel(NAME)
    c.onmessage = (m) => { if (m.data === 'logout') cb() }
    return () => c.close()
  } catch (e) {
    return () => {}
  }
}
