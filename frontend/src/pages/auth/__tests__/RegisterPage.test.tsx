import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { RegisterPage } from '@/pages/auth/RegisterPage'
import { useAuthStore } from '@/store/auth'

vi.mock('@/lib/api', () => ({
  default: { post: vi.fn() },
}))

function renderRegister() {
  return render(
    <MemoryRouter initialEntries={['/auth/register']}>
      <Routes>
        <Route path="/auth/register" element={<RegisterPage />} />
        <Route path="/auth/login" element={<div>Login Page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

async function fillAndSubmitForm(user: ReturnType<typeof userEvent.setup>, overrides: Partial<{
  fullName: string
  username: string
  email: string
  password: string
  confirmPassword: string
}> = {}) {
  const defaults = {
    fullName: 'Test User',
    username: 'testuser',
    email: 'test@example.com',
    password: 'password123',
    confirmPassword: 'password123',
    ...overrides,
  }
  await user.type(screen.getByLabelText(/full name/i), defaults.fullName)
  await user.type(screen.getByLabelText(/^username$/i), defaults.username)
  await user.type(screen.getByLabelText(/email/i), defaults.email)
  await user.type(screen.getByLabelText(/^password$/i), defaults.password)
  await user.type(screen.getByLabelText(/confirm password/i), defaults.confirmPassword)
  await user.click(screen.getByRole('button', { name: /create account/i }))
}

describe('RegisterPage', () => {
  beforeEach(() => {
    useAuthStore.setState({ isLoading: false, error: null, register: vi.fn() })
    vi.clearAllMocks()
  })

  it('renders all required form fields', () => {
    renderRegister()
    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^username$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/confirm password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
  })

  it('renders page title', () => {
    renderRegister()
    // "Create Account" appears in both heading and button; select the heading specifically
    expect(screen.getByRole('heading', { name: 'Create Account' })).toBeInTheDocument()
  })

  it('shows link to login page', () => {
    renderRegister()
    expect(screen.getByRole('link', { name: /sign in/i })).toBeInTheDocument()
  })

  it('shows error when passwords do not match', async () => {
    const user = userEvent.setup()
    renderRegister()
    await fillAndSubmitForm(user, { password: 'password123', confirmPassword: 'different' })
    expect(screen.getByText('Passwords do not match')).toBeInTheDocument()
  })

  it('shows error when password is too short', async () => {
    const user = userEvent.setup()
    renderRegister()
    await fillAndSubmitForm(user, { password: 'short', confirmPassword: 'short' })
    expect(screen.getByText('Password must be at least 8 characters')).toBeInTheDocument()
  })

  it('does not call register when passwords do not match', async () => {
    const user = userEvent.setup()
    const registerMock = vi.fn()
    useAuthStore.setState({ isLoading: false, error: null, register: registerMock })
    renderRegister()
    await fillAndSubmitForm(user, { password: 'pass1234', confirmPassword: 'pass5678' })
    expect(registerMock).not.toHaveBeenCalled()
  })

  it('calls register with correct data on valid submission', async () => {
    const user = userEvent.setup()
    const registerMock = vi.fn().mockResolvedValue({ userId: 'usr-1' })
    useAuthStore.setState({ isLoading: false, error: null, register: registerMock })
    renderRegister()
    await fillAndSubmitForm(user)

    await waitFor(() => {
      expect(registerMock).toHaveBeenCalledWith(
        expect.objectContaining({
          username: 'testuser',
          email: 'test@example.com',
          password: 'password123',
          fullName: 'Test User',
        }),
      )
    })
  })

  it('navigates to login page after successful registration', async () => {
    const user = userEvent.setup()
    const registerMock = vi.fn().mockResolvedValue({ userId: 'usr-1' })
    useAuthStore.setState({ isLoading: false, error: null, register: registerMock })
    renderRegister()
    await fillAndSubmitForm(user)

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument()
    })
  })

  it('displays error from store', () => {
    useAuthStore.setState({ isLoading: false, error: 'Email already taken', register: vi.fn() })
    renderRegister()
    expect(screen.getByText('Email already taken')).toBeInTheDocument()
  })

  it('disables button while loading', () => {
    useAuthStore.setState({ isLoading: true, error: null, register: vi.fn() })
    renderRegister()
    expect(screen.getByRole('button', { name: /create account/i })).toBeDisabled()
  })
})
