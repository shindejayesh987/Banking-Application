import { test, expect } from '@playwright/test'

// Use regex patterns so route mocking works regardless of which origin or port
// the app uses for API calls (VITE_API_BASE_URL may or may not be set).
const ROUTE = {
  login:    /\/auth\/login(\?.*)?$/,
  register: /\/auth\/register(\?.*)?$/,
  logout:   /\/auth\/logout(\?.*)?$/,
}

const fakeAuth = {
  accessToken: 'test-access-token',
  refreshToken: 'test-refresh-token',
  userId: 'user-123',
  username: 'testuser',
}

// ──────────────────────────────────────────────────────────────────────────────
// Login page
// ──────────────────────────────────────────────────────────────────────────────

test.describe('Login page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/auth/login')
  })

  test('shows the login form with all required elements', async ({ page }) => {
    await expect(page.getByText('Banking System Design Lab')).toBeVisible()
    await expect(page.getByText('Sign in to your account')).toBeVisible()
    await expect(page.getByLabel('Username')).toBeVisible()
    await expect(page.getByLabel('Password')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Sign In' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'Register' })).toBeVisible()
  })

  test('navigates to register page via Register link', async ({ page }) => {
    await page.getByRole('link', { name: 'Register' }).click()
    await expect(page).toHaveURL('/auth/register')
  })

  test('shows error message on failed login', async ({ page }) => {
    await page.route(ROUTE.login, (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid credentials' }),
      }),
    )

    await page.getByLabel('Username').fill('wronguser')
    await page.getByLabel('Password').fill('wrongpass')
    await page.getByRole('button', { name: 'Sign In' }).click()

    await expect(page.getByText('Invalid credentials')).toBeVisible()
  })

  test('disables the submit button while login is in-flight', async ({ page }) => {
    let resolveRequest!: () => void
    await page.route(ROUTE.login, async (route) => {
      await new Promise<void>((r) => { resolveRequest = r })
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fakeAuth) })
    })

    await page.getByLabel('Username').fill('testuser')
    await page.getByLabel('Password').fill('password123')

    const button = page.getByRole('button', { name: 'Sign In' })
    await button.click()

    // While the request is pending the button should be disabled
    await expect(button).toBeDisabled()

    // Let the request finish and wait for navigation
    resolveRequest()
    await page.waitForURL('/')
  })

  test('redirects to dashboard on successful login', async ({ page }) => {
    await page.route(ROUTE.login, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(fakeAuth) }),
    )

    await page.getByLabel('Username').fill('testuser')
    await page.getByLabel('Password').fill('password123')
    await page.getByRole('button', { name: 'Sign In' }).click()

    await page.waitForURL('/')
    await expect(page).toHaveURL('/')
  })
})

// ──────────────────────────────────────────────────────────────────────────────
// Register page
// ──────────────────────────────────────────────────────────────────────────────

test.describe('Register page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/auth/register')
  })

  test('shows registration form with all fields', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Create Account' })).toBeVisible()
    await expect(page.getByLabel('Full Name')).toBeVisible()
    await expect(page.getByLabel('Username')).toBeVisible()
    await expect(page.getByLabel('Email')).toBeVisible()
    await expect(page.getByLabel('Password', { exact: true })).toBeVisible()
    await expect(page.getByLabel('Confirm Password')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Create Account' })).toBeVisible()
  })

  test('shows validation error for password mismatch', async ({ page }) => {
    await page.getByLabel('Full Name').fill('Test User')
    await page.getByLabel('Username').fill('testuser')
    await page.getByLabel('Email').fill('test@example.com')
    await page.getByLabel('Password', { exact: true }).fill('password123')
    await page.getByLabel('Confirm Password').fill('different456')
    await page.getByRole('button', { name: 'Create Account' }).click()

    await expect(page.getByText('Passwords do not match')).toBeVisible()
  })

  test('shows validation error for short password', async ({ page }) => {
    await page.getByLabel('Full Name').fill('Test User')
    await page.getByLabel('Username').fill('testuser')
    await page.getByLabel('Email').fill('test@example.com')
    await page.getByLabel('Password', { exact: true }).fill('1234567')
    await page.getByLabel('Confirm Password').fill('1234567')
    await page.getByRole('button', { name: 'Create Account' }).click()

    await expect(page.getByText('Password must be at least 8 characters')).toBeVisible()
  })

  test('navigates to login on successful registration', async ({ page }) => {
    await page.route(ROUTE.register, (route) =>
      route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ userId: 'new-user-id', username: 'newuser' }),
      }),
    )

    await page.getByLabel('Full Name').fill('New User')
    await page.getByLabel('Username').fill('newuser')
    await page.getByLabel('Email').fill('newuser@example.com')
    await page.getByLabel('Password', { exact: true }).fill('securepassword')
    await page.getByLabel('Confirm Password').fill('securepassword')
    await page.getByRole('button', { name: 'Create Account' }).click()

    await page.waitForURL('/auth/login')
    await expect(page).toHaveURL('/auth/login')
  })

  test('shows API error on registration failure', async ({ page }) => {
    await page.route(ROUTE.register, (route) =>
      route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Email already registered' }),
      }),
    )

    await page.getByLabel('Full Name').fill('Existing User')
    await page.getByLabel('Username').fill('existinguser')
    await page.getByLabel('Email').fill('existing@example.com')
    await page.getByLabel('Password', { exact: true }).fill('password123')
    await page.getByLabel('Confirm Password').fill('password123')
    await page.getByRole('button', { name: 'Create Account' }).click()

    await expect(page.getByText('Email already registered')).toBeVisible()
  })

  test('has a link back to login', async ({ page }) => {
    await page.getByRole('link', { name: 'Sign in' }).click()
    await expect(page).toHaveURL('/auth/login')
  })
})

// ──────────────────────────────────────────────────────────────────────────────
// Protected routes (unauthenticated)
// ──────────────────────────────────────────────────────────────────────────────

test.describe('Protected routes', () => {
  test('redirects unauthenticated user from / to login page', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL('/auth/login')
  })

  test('redirects unauthenticated user from /accounts to login page', async ({ page }) => {
    await page.goto('/accounts')
    await expect(page).toHaveURL('/auth/login')
  })
})

// ──────────────────────────────────────────────────────────────────────────────
// Authenticated dashboard
// ──────────────────────────────────────────────────────────────────────────────

test.describe('Dashboard (authenticated)', () => {
  test.beforeEach(async ({ page }) => {
    // Inject auth tokens before navigation so ProtectedRoute sees them
    await page.addInitScript((auth) => {
      localStorage.setItem('accessToken', auth.accessToken)
      localStorage.setItem('refreshToken', auth.refreshToken)
      localStorage.setItem('userId', auth.userId)
      localStorage.setItem('username', auth.username)
    }, fakeAuth)
  })

  test('shows dashboard heading and 6 system design cards', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()

    // Each card title is an <h3>; use heading role to avoid strict-mode violations
    // (the same text also appears in the sidebar navigation links)
    const cards = ['Redis Cache', 'Kafka', 'Circuit Breaker', 'Rate Limiter', 'DB Replication', 'Saga']
    for (const name of cards) {
      await expect(page.getByRole('heading', { name, exact: true })).toBeVisible()
    }
  })

  test('sidebar shows username', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText('testuser')).toBeVisible()
  })

  test('sidebar has Accounts and New Transfer navigation links', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('link', { name: 'Accounts' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'New Transfer' })).toBeVisible()
  })

  test('navigating to /accounts shows placeholder page', async ({ page }) => {
    await page.goto('/accounts')
    await expect(page.getByText('This page will be built in a future phase.')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Accounts' })).toBeVisible()
  })

  test('navigating to /system/redis shows placeholder page with title', async ({ page }) => {
    await page.goto('/system/redis')
    await expect(page.getByText('This page will be built in a future phase.')).toBeVisible()
    await expect(page.getByText('System > Redis')).toBeVisible()
  })

  test('logout button clears session and redirects to login', async ({ page }) => {
    await page.route(ROUTE.logout, (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: '{}' }),
    )

    await page.goto('/')
    // The logout button is the only <button> inside the <aside> sidebar
    await page.locator('aside button').click()

    await expect(page).toHaveURL('/auth/login')

    const token = await page.evaluate(() => localStorage.getItem('accessToken'))
    expect(token).toBeNull()
  })
})
