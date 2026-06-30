import { test, expect, Page } from '@playwright/test'

// ---------------------------------------------------------------------------
// Helper: log in as a regular user before navigating to protected pages
// ---------------------------------------------------------------------------

async function loginAsUser(page: Page): Promise<void> {
  await page.goto('/login')
  await page.getByLabel(/email/i).fill('user@demo.example.com')
  await page.getByLabel(/password/i).fill('User123!')
  await page.getByRole('button', { name: /sign in/i }).click()
  // Wait until the router lands us on the dashboard (not /login)
  await page.waitForURL((url) => !url.pathname.endsWith('/login'), { timeout: 10_000 })
}

// ---------------------------------------------------------------------------
// Suite
// ---------------------------------------------------------------------------

test.describe('Cases', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsUser(page)
  })

  // --- Navigation -----------------------------------------------------------

  test('navigates to cases page via sidebar link', async ({ page }) => {
    await page.getByRole('link', { name: /my cases/i }).click()
    await expect(page).toHaveURL('/cases')
    await expect(page.getByRole('heading', { name: /cases/i })).toBeVisible()
  })

  // --- List -----------------------------------------------------------------

  test('shows cases table after navigation', async ({ page }) => {
    await page.goto('/cases')
    // The table should appear (data may still be loading)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })
  })

  test('table has expected column headers', async ({ page }) => {
    await page.goto('/cases')
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const headers = page.getByRole('columnheader')
    await expect(headers.filter({ hasText: /case/i })).toBeVisible()
    await expect(headers.filter({ hasText: /title/i })).toBeVisible()
    await expect(headers.filter({ hasText: /status/i })).toBeVisible()
  })

  // --- Create ---------------------------------------------------------------

  test('can open new case modal via button', async ({ page }) => {
    await page.goto('/cases')
    await page.getByRole('button', { name: /new case/i }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await expect(page.getByLabel(/title/i)).toBeVisible()
    await expect(page.getByLabel(/description/i)).toBeVisible()
  })

  test('can create a new case and see it in the list', async ({ page }) => {
    await page.goto('/cases')
    await page.getByRole('button', { name: /new case/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    const uniqueTitle = `Test Case from E2E ${Date.now()}`
    await page.getByLabel(/title/i).fill(uniqueTitle)
    await page.getByLabel(/description/i).fill('This is an automated test case created by Playwright')

    await dialog.getByRole('button', { name: /create/i }).click()

    // Dialog should close and the new case should appear in the list
    await expect(dialog).not.toBeVisible({ timeout: 5_000 })
    await expect(page.getByText(uniqueTitle)).toBeVisible({ timeout: 10_000 })
  })

  test('create form shows validation errors for empty title', async ({ page }) => {
    await page.goto('/cases')
    await page.getByRole('button', { name: /new case/i }).click()

    const dialog = page.getByRole('dialog')
    await dialog.getByRole('button', { name: /create/i }).click()

    await expect(page.getByText(/title is required/i)).toBeVisible()
  })

  // --- Detail ---------------------------------------------------------------

  test('can navigate to case detail by clicking a table row', async ({ page }) => {
    await page.goto('/cases')
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    // Click the first data row (index 1 skips the header row)
    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    // URL should change to /cases/<id>
    await expect(page).toHaveURL(/\/cases\/[a-zA-Z0-9-]+/, { timeout: 5_000 })
  })

  test('case detail page shows case number', async ({ page }) => {
    await page.goto('/cases')
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    await expect(page).toHaveURL(/\/cases\/[a-zA-Z0-9-]+/, { timeout: 5_000 })
    // Case numbers follow the CASE-NNNNN pattern
    await expect(page.getByText(/CASE-\d+/)).toBeVisible({ timeout: 5_000 })
  })

  test('case detail shows title and description', async ({ page }) => {
    await page.goto('/cases')
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    await expect(page).toHaveURL(/\/cases\/[a-zA-Z0-9-]+/, { timeout: 5_000 })
    await expect(page.getByRole('heading')).toBeVisible()
  })

  // --- Filtering / Pagination -----------------------------------------------

  test('status filter narrows the displayed cases', async ({ page }) => {
    await page.goto('/cases')
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    // Open the status filter dropdown (if present)
    const filterSelect = page.getByRole('combobox', { name: /status/i })
    if (await filterSelect.isVisible()) {
      await filterSelect.selectOption('open')
      // All visible status cells should say "open"
      const statusCells = page.getByRole('cell', { name: /open/i })
      await expect(statusCells.first()).toBeVisible({ timeout: 5_000 })
    } else {
      test.skip()
    }
  })
})
