import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  // No custom base: build files land in /assets/ (Nuxt 2 uses /_nuxt/, so they never collide).
  // The gateway routes /assets/* to this app. Page URLs are identical to legacy.
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  server: { port: 5173 },
})
