import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { LoginPage } from '@/pages/auth/LoginPage'
import { useAuthStore } from '@/store/auth'

vi.mock('@/lib/api', () => ({
  default: { post: vi.fn() },
}))

function renderLogin(initialPath = '/auth/login') {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/auth/login" element={<LoginPage />} />
        <Route path="/auth/register" element={<div>Register Page</div>} />
        <Route path="/" element={<div>Home Page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('LoginPage', () => {
  beforeEach(() => {
    useAuthStore.setState({
      isAuthenticated: false,
      isLoading: false,
      error: null,
      login: vi.fn(),
    })
    vi.clearAllMocks()
  })

  it('renders the login form with username and password fields', () => {
    renderLogin()
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('renders the page title', () => {
    renderLogin()
    expect(screen.getByText('Banking System Design Lab')).toBeInTheDocument()
  })

  it('shows link to register page', () => {
    renderLogin()
    expect(screen.getByRole('link', { name: /register/i })).toBeInTheDocument()
  })

  it('submits credentials when form is submitted', async () => {
    const user = userEvent.setup()
    const loginMock = vi.fn().mockResolvedValue(undefined)
    useAuthStore.setState({ isLoading: false, error: null, login: loginMock })

    renderLogin()

    await user.type(screen.getByLabelText(/username/i), 'alice')
    await user.type(screen.getByLabelText(/password/i), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(loginMock).toHaveBeenCalledWith({ username: 'alice', password: 'password123' })
    })
  })

  it('navigates to home on successful login', async () => {
    const user = userEvent.setup()
    const loginMock = vi.fn().mockResolvedValue(undefined)
    useAuthStore.setState({ isLoading: false, error: null, login: loginMock })

    renderLogin()

    await user.type(screen.getByLabelText(/username/i), 'alice')
    await user.type(screen.getByLabelText(/password/i), 'secret')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText('Home Page')).toBeInTheDocument()
    })
  })

  it('displays error alert when store has error', () => {
    useAuthStore.setState({ isLoading: false, error: 'Invalid credentials', login: vi.fn() })
    renderLogin()
    expect(screen.getByText('Invalid credentials')).toBeInTheDocument()
  })

  it('disables submit button while loading', () => {
    useAuthStore.setState({ isLoading: true, error: null, login: vi.fn() })
    renderLogin()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeDisabled()
  })

  it('button is enabled when not loading', () => {
    useAuthStore.setState({ isLoading: false, error: null, login: vi.fn() })
    renderLogin()
    expect(screen.getByRole('button', { name: /sign in/i })).not.toBeDisabled()
  })
})
