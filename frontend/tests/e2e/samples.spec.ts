import { test, expect } from '@playwright/test'
import { toolUI } from '../../src/toolsMeta'

// Every tool that ships a "Use sample" button must decode its own sample with
// the default control values (plus its sampleControls). This guards against
// samples drifting out of sync with tool defaults or backend behavior.
const sampledTools = Object.entries(toolUI).filter(([, ui]) => ui.sample)

test('every tool sample decodes without error', async ({ page }) => {
  await page.goto('/')
  await expect(page.locator('.sidebar .nav-item').first()).toBeVisible()

  const failures: string[] = []
  for (const [name] of sampledTools) {
    // Navigate via the API-provided sidebar (titles come from the backend).
    const res = await page.evaluate(async (tool) => {
      const meta = await (await fetch('/api/v1/tools')).json()
      return meta.find((t: { name: string }) => t.name === tool)?.title ?? null
    }, name)
    if (!res) {
      failures.push(`${name}: not in backend tool list`)
      continue
    }
    await page.locator('.sidebar .nav-item', { hasText: new RegExp(`^${res.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`) }).first().click()
    await page.getByRole('button', { name: 'Use sample' }).click()
    // Debounced run + request round trip.
    await page.waitForTimeout(700)
    const hasError = await page.locator('.error').isVisible()
    const hasResult = await page.locator('.card:not(.input-card)').isVisible()
    if (hasError) {
      failures.push(`${name}: ${await page.locator('.error').innerText()}`)
    } else if (!hasResult) {
      failures.push(`${name}: no result rendered`)
    }
  }
  expect(failures, failures.join('\n')).toEqual([])
})
