import { test, expect } from '@playwright/test'

test('decodes JSON locally without leaving the origin', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('security-toolbox')).toBeVisible()

  // Pick the JSON formatter from the sidebar.
  await page.getByRole('button', { name: 'JSON Formatter' }).click()
  await page.locator('textarea').fill('{"a":1,"b":[2,3]}')

  // The result panel should show the parsed value as a collapsible tree.
  await expect(page.locator('.jtree').first()).toBeVisible()
  await expect(page.locator('.jtree').first()).toContainText('a')
  await expect(page.locator('.jtree').first()).toContainText('1')
})

test('reports a precise error for invalid JSON', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: 'JSON Formatter' }).click()
  await page.locator('textarea').fill('{"a":}')
  await expect(page.locator('.error')).toContainText('line 1')
})
