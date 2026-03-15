import { describe, it, expect, beforeEach, vi } from 'vitest'

// Mock the api module before importing the store
vi.mock('@/lib/api', () => ({
  default: {
    post: vi.fn(),
  },
}))

import api from '@/lib/api'
import { useAuthStore } from '@/store/auth'

const mockApi = api as unknown as { post: ReturnType<typeof vi.fn> }

describe('useAuthStore', () => {
  beforeEach(() => {
    // Reset store state between tests
    useAuthStore.setState({
      userId: null,
      username: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
    })
    localStorage.clear()
    vi.clearAllMocks()
  })

  // --- hydrate ---

  it('hydrate restores auth state from localStorage when token and userId exist', () => {
    localStorage.setItem('accessToken', 'tok-1')
    localStorage.setItem('userId', 'usr-1')
    localStorage.setItem('username', 'alice')

    useAuthStore.getState().hydrate()

    const { isAuthenticated, userId, username } = useAuthStore.getState()
    expect(isAuthenticated).toBe(true)
    expect(userId).toBe('usr-1')
    expect(username).toBe('alice')
  })

  it('hydrate does nothing when accessToken is missing', () => {
    localStorage.setItem('userId', 'usr-1')
    useAuthStore.getState().hydrate()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('hydrate does nothing when userId is missing', () => {
    localStorage.setItem('accessToken', 'tok-1')
    useAuthStore.getState().hydrate()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  // --- login ---

  it('login sets isAuthenticated and stores tokens on success', async () => {
    mockApi.post.mockResolvedValueOnce({
      data: {
        accessToken: 'at-1',
        refreshToken: 'rt-1',
        userId: 'usr-1',
        username: 'alice',
      },
    })

    await useAuthStore.getState().login({ username: 'alice', password: '1234' })

    expect(useAuthStore.getState().isAuthenticated).toBe(true)
    expect(useAuthStore.getState().userId).toBe('usr-1')
    expect(useAuthStore.getState().username).toBe('alice')
    expect(useAuthStore.getState().isLoading).toBe(false)
    expect(localStorage.getItem('accessToken')).toBe('at-1')
    expect(localStorage.getItem('refreshToken')).toBe('rt-1')
    expect(localStorage.getItem('userId')).toBe('usr-1')
    expect(localStorage.getItem('username')).toBe('alice')
  })

  it('login sets error on failure', async () => {
    mockApi.post.mockRejectedValueOnce({
      response: { data: { error: 'Invalid credentials' } },
    })

    await expect(useAuthStore.getState().login({ username: 'bad', password: 'bad' })).rejects.toBeDefined()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
    expect(useAuthStore.getState().error).toBe('Invalid credentials')
    expect(useAuthStore.getState().isLoading).toBe(false)
  })

  it('login includes lockout info in error message', async () => {
    mockApi.post.mockRejectedValueOnce({
      response: {
        data: {
          error: 'Account locked',
          lockedUntil: new Date('2099-01-01T10:00:00Z').toISOString(),
        },
      },
    })

    await expect(useAuthStore.getState().login({ username: 'locked', password: '1234' })).rejects.toBeDefined()

    const error = useAuthStore.getState().error ?? ''
    expect(error).toContain('Account locked')
    expect(error).toContain('locked until')
  })

  it('login uses fallback error message when response has no error field', async () => {
    mockApi.post.mockRejectedValueOnce({})

    await expect(useAuthStore.getState().login({ username: 'x', password: 'y' })).rejects.toBeDefined()

    expect(useAuthStore.getState().error).toBe('Login failed')
  })

  // --- register ---

  it('register returns response data on success', async () => {
    mockApi.post.mockResolvedValueOnce({
      data: { userId: 'usr-new', email: 'new@example.com' },
    })

    const result = await useAuthStore.getState().register({
      username: 'new',
      email: 'new@example.com',
      password: 'password123',
      fullName: 'New User',
    })

    expect(result.userId).toBe('usr-new')
    expect(useAuthStore.getState().isLoading).toBe(false)
  })

  it('register sets error on failure', async () => {
    mockApi.post.mockRejectedValueOnce({
      response: { data: { error: 'Email already taken' } },
    })

    await expect(useAuthStore.getState().register({
      username: 'dup',
      email: 'dup@example.com',
      password: 'pass',
      fullName: 'Dup',
    })).rejects.toBeDefined()

    expect(useAuthStore.getState().error).toBe('Email already taken')
  })

  // --- logout ---

  it('logout clears localStorage and resets state', async () => {
    // Set up logged-in state
    localStorage.setItem('accessToken', 'tok')
    localStorage.setItem('refreshToken', 'rtok')
    localStorage.setItem('userId', 'usr-1')
    localStorage.setItem('username', 'alice')
    useAuthStore.setState({ isAuthenticated: true, userId: 'usr-1', username: 'alice' })

    mockApi.post.mockResolvedValueOnce({})

    await useAuthStore.getState().logout()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
    expect(useAuthStore.getState().userId).toBeNull()
    expect(useAuthStore.getState().username).toBeNull()
    expect(localStorage.getItem('accessToken')).toBeNull()
    expect(localStorage.getItem('refreshToken')).toBeNull()
  })

  it('logout clears state even when API call fails', async () => {
    localStorage.setItem('accessToken', 'tok')
    useAuthStore.setState({ isAuthenticated: true, userId: 'usr-1', username: 'alice' })

    mockApi.post.mockRejectedValueOnce(new Error('network error'))

    await useAuthStore.getState().logout()

    expect(useAuthStore.getState().isAuthenticated).toBe(false)
    expect(localStorage.getItem('accessToken')).toBeNull()
  })
})
