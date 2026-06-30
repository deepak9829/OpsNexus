import { test, expect, Page } from '@playwright/test'

const ADMIN_URL = 'http://localhost:3001'

// ---------------------------------------------------------------------------
// Helper: log in as the platform administrator
// ---------------------------------------------------------------------------

async function loginAsAdmin(page: Page): Promise<void> {
  await page.goto(`${ADMIN_URL}/login`)
  await page.getByLabel(/email/i).fill('admin@opsnexus.com')
  await page.getByLabel(/password/i).fill('Admin123!')
  await page.getByRole('button', { name: /sign in/i }).click()
  // Wait until the router lands us on the admin dashboard (not /login)
  await page.waitForURL(
    (url) => url.href.startsWith(ADMIN_URL) && !url.pathname.endsWith('/login'),
    { timeout: 10_000 },
  )
}

// ---------------------------------------------------------------------------
// Suite: Admin portal — Tenants section
// ---------------------------------------------------------------------------

test.describe('Admin: Tenants', () => {
  // Override the baseURL to point at the admin portal for this describe block
  test.use({ baseURL: ADMIN_URL })

  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page)
  })

  // --- List -----------------------------------------------------------------

  test('tenants list page loads with a heading', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('heading', { name: /tenants/i })).toBeVisible({ timeout: 10_000 })
  })

  test('tenants table is visible after navigation', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })
  })

  test('tenants table has expected columns', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const headers = page.getByRole('columnheader')
    await expect(headers.filter({ hasText: /name/i })).toBeVisible()
    await expect(headers.filter({ hasText: /status/i })).toBeVisible()
  })

  // --- Create ---------------------------------------------------------------

  test('can open new tenant modal via button', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await page.getByRole('button', { name: /new tenant/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(page.getByLabel(/tenant name/i)).toBeVisible()
  })

  test('new tenant modal contains required form fields', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await page.getByRole('button', { name: /new tenant/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    // Should have at minimum: tenant name, admin email
    await expect(page.getByLabel(/tenant name/i)).toBeVisible()
    await expect(page.getByLabel(/admin email/i)).toBeVisible()
  })

  test('create tenant form shows validation errors for empty submission', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await page.getByRole('button', { name: /new tenant/i }).click()

    const dialog = page.getByRole('dialog')
    await dialog.getByRole('button', { name: /create/i }).click()

    // At least the tenant name error should be visible
    await expect(page.getByText(/tenant name is required/i)).toBeVisible()
  })

  test('can dismiss new tenant modal with cancel', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await page.getByRole('button', { name: /new tenant/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()

    // Close via the Cancel button or Escape key
    const cancelButton = dialog.getByRole('button', { name: /cancel/i })
    if (await cancelButton.isVisible()) {
      await cancelButton.click()
    } else {
      await page.keyboard.press('Escape')
    }

    await expect(dialog).not.toBeVisible({ timeout: 3_000 })
  })

  // --- Detail ---------------------------------------------------------------

  test('clicking a tenant row navigates to its detail page', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    await expect(page).toHaveURL(/\/tenants\/[a-zA-Z0-9-]+/, { timeout: 5_000 })
  })

  test('tenant detail page shows tenant name heading', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    await expect(page).toHaveURL(/\/tenants\/[a-zA-Z0-9-]+/, { timeout: 5_000 })
    // The detail page should display the tenant's name in a heading
    await expect(page.getByRole('heading')).toBeVisible()
  })

  // --- Status toggle --------------------------------------------------------

  test('tenant detail page has a suspend/activate toggle', async ({ page }) => {
    await page.goto(`${ADMIN_URL}/tenants`)
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10_000 })

    const firstRow = page.getByRole('row').nth(1)
    await firstRow.click()

    await expect(page).toHaveURL(/\/tenants\/[a-zA-Z0-9-]+/, { timeout: 5_000 })

    // The page should offer either a "Suspend" or "Activate" action
    const suspendButton = page.getByRole('button', { name: /suspend/i })
    const activateButton = page.getByRole('button', { name: /activate/i })

    const hasToggle = (await suspendButton.isVisible()) || (await activateButton.isVisible())
    expect(hasToggle).toBe(true)
  })
})
