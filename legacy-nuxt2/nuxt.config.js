export default {
  // Universal mode (SSR + client hydration), started with `npm start` (nuxt start) like production.
  ssr: true,
  target: 'server',
  telemetry: false,
  head: {
    title: 'ApplianceHub POS (legacy)',
    meta: [
      { charset: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
    ],
  },
  css: ['~/assets/pos.css'],
  plugins: [
    '~/plugins/api.js',                                      // $api: axios + cookie forwarding (SSR) + CSRF + silent refresh
    { src: '~/plugins/hard-navigation.js', mode: 'client' }, // crossing into a Vue 3 route = full page load
  ],
  // Global route middleware: loads session (/api/me) + cart, redirects to /login when unauthenticated
  router: { middleware: ['auth'] },
  // Server-only: SSR talks to the API container directly (not through the gateway)
  privateRuntimeConfig: { apiInternalUrl: process.env.API_INTERNAL_URL || 'http://localhost:8080' },
  server: { host: '0.0.0.0', port: 3000 },
}
