import { test, expect } from '@playwright/test'

test.describe('Authentication', () => {
  test('login page renders correctly', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: /opsnexus/i })).toBeVisible()
    await expect(page.getByLabel(/email/i)).toBeVisible()
    await expect(page.getByLabel(/password/i)).toBeVisible()
    await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible()
  })

  test('shows validation errors on empty submit', async ({ page }) => {
    await page.goto('/login')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByText(/valid email required/i)).toBeVisible()
    await expect(page.getByText(/password must be/i)).toBeVisible()
  })

  test('shows error on invalid credentials', async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel(/email/i).fill('bad@example.com')
    await page.getByLabel(/password/i).fill('wrongpassword')
    await page.getByRole('button', { name: /sign in/i }).click()
    // The API returns 401; the UI should render an alert with the error message
    await expect(page.getByRole('alert')).toBeVisible()
  })

  test('redirects unauthenticated user to login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
  })

  test('email field accepts valid email format', async ({ page }) => {
    await page.goto('/login')
    const emailInput = page.getByLabel(/email/i)
    await emailInput.fill('not-an-email')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByText(/valid email required/i)).toBeVisible()
  })

  test('password field is masked', async ({ page }) => {
    await page.goto('/login')
    const passwordInput = page.getByLabel(/password/i)
    await expect(passwordInput).toHaveAttribute('type', 'password')
  })

  test('sign in button is disabled while request is in flight', async ({ page }) => {
    await page.goto('/login')

    // Intercept the login API call and delay it so we can assert the loading state
    await page.route('**/api/v1/auth/login', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 500))
      await route.continue()
    })

    await page.getByLabel(/email/i).fill('user@demo.example.com')
    await page.getByLabel(/password/i).fill('User123!')

    const signInButton = page.getByRole('button', { name: /sign in/i })
    await signInButton.click()

    // Button should be disabled (or show a spinner) during the pending request
    await expect(signInButton).toBeDisabled()
  })
})
