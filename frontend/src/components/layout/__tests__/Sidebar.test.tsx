import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { Sidebar } from '@/components/layout/Sidebar'
import { useAuthStore } from '@/store/auth'

vi.mock('@/lib/api', () => ({
  default: { post: vi.fn().mockResolvedValue({}) },
}))

function renderSidebar(username = 'alice') {
  useAuthStore.setState({ isAuthenticated: true, username })
  return render(
    <MemoryRouter>
      <Sidebar />
    </MemoryRouter>,
  )
}

describe('Sidebar', () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: true, username: 'alice' })
  })

  it('renders the app title', () => {
    renderSidebar()
    expect(screen.getByText('Banking Lab')).toBeInTheDocument()
  })

  it('renders Dashboard nav item', () => {
    renderSidebar()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders Accounts nav item', () => {
    renderSidebar()
    expect(screen.getByText('Accounts')).toBeInTheDocument()
  })

  it('renders System Design heading', () => {
    renderSidebar()
    expect(screen.getByText('System Design')).toBeInTheDocument()
  })

  it('renders system design nav items', () => {
    renderSidebar()
    expect(screen.getByText('Redis Cache')).toBeInTheDocument()
    expect(screen.getByText('Kafka')).toBeInTheDocument()
    expect(screen.getByText('Circuit Breaker')).toBeInTheDocument()
    expect(screen.getByText('Rate Limiting')).toBeInTheDocument()
    expect(screen.getByText('DB Replication')).toBeInTheDocument()
    expect(screen.getByText('Saga')).toBeInTheDocument()
    expect(screen.getByText('Audit Log')).toBeInTheDocument()
  })

  it('displays the logged-in username', () => {
    renderSidebar('bobsmith')
    expect(screen.getByText('bobsmith')).toBeInTheDocument()
  })

  it('calls logout when logout button is clicked', async () => {
    const user = userEvent.setup()
    const logoutSpy = vi.fn().mockResolvedValue(undefined)
    useAuthStore.setState({ logout: logoutSpy, isAuthenticated: true, username: 'alice' })

    render(
      <MemoryRouter>
        <Sidebar />
      </MemoryRouter>,
    )

    // Find the logout button by its aria role (button with LogOut icon)
    const buttons = screen.getAllByRole('button')
    await user.click(buttons[0]) // only one button in sidebar
    expect(logoutSpy).toHaveBeenCalledOnce()
  })
})
