import { defineConfig } from '@playwright/test'

// These e2e tests assume the full app (Go binary serving the built SPA) is
// running on host port 8075 (docker-compose maps 8075 -> container 8080).
// Start it with `docker compose up` before `npm run test:e2e`.
export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:8075',
    headless: true
  }
})
