// In the real project this lives in packages/nav-config and is shared by both shells.
export const MENU = [
  { to: '/', label: 'Dashboard', perm: 'orders.read' },
  { to: '/pos', label: 'POS / Checkout', perm: 'cart.use' },
  { to: '/products', label: 'Products', perm: 'products.read' },
  { to: '/orders', label: 'Orders', perm: 'orders.read' },
  { to: '/settings', label: 'Settings', perm: 'settings.read' },
  { to: '/settings/protection-plan', label: 'Protection Plans', perm: 'cart.use' },
]
