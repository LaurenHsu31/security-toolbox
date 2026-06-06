import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// During `npm run dev` the Go backend runs on :8080; proxy the API to it.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // No inline scripts -> compatible with the strict `script-src 'self'` CSP.
    modulePreload: { polyfill: false }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080'
    }
  }
})
