import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from '@/components/layout/ProtectedRoute'
import { useAuthStore } from '@/store/auth'

function renderWithRouter(initialPath: string, isAuthenticated: boolean) {
  useAuthStore.setState({ isAuthenticated })

  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <div>Protected Content</div>
            </ProtectedRoute>
          }
        />
        <Route path="/auth/login" element={<div>Login Page</div>} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: false })
  })

  it('renders children when authenticated', () => {
    renderWithRouter('/', true)
    expect(screen.getByText('Protected Content')).toBeInTheDocument()
  })

  it('redirects to /auth/login when not authenticated', () => {
    renderWithRouter('/', false)
    expect(screen.getByText('Login Page')).toBeInTheDocument()
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument()
  })

  it('does not render children when not authenticated', () => {
    renderWithRouter('/', false)
    expect(screen.queryByText('Protected Content')).toBeNull()
  })
})
