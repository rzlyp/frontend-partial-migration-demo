// Cross-tab logout (same origin, so it also reaches the legacy app in another tab).
const NAME = 'pos-auth'

export function broadcastLogout(): void {
  try {
    const c = new BroadcastChannel(NAME)
    c.postMessage('logout')
    c.close()
  } catch {
    /* unsupported */
  }
}

export function onLogout(cb: () => void): () => void {
  try {
    const c = new BroadcastChannel(NAME)
    c.onmessage = (m) => {
      if (m.data === 'logout') cb()
    }
    return () => c.close()
  } catch {
    return () => {}
  }
}
