import { create } from 'zustand'
import api from '@/lib/api'
import type { LoginRequest, LoginResponse, RegisterRequest, RegisterResponse } from '@/types'

interface AuthState {
  userId: string | null
  username: string | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
  login: (data: LoginRequest) => Promise<void>
  register: (data: RegisterRequest) => Promise<RegisterResponse>
  logout: () => Promise<void>
  hydrate: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  userId: null,
  username: null,
  isAuthenticated: false,
  isLoading: false,
  error: null,

  hydrate: () => {
    const token = localStorage.getItem('accessToken')
    const userId = localStorage.getItem('userId')
    const username = localStorage.getItem('username')
    if (token && userId) {
      set({ isAuthenticated: true, userId, username })
    }
  },

  login: async (data: LoginRequest) => {
    set({ isLoading: true, error: null })
    try {
      const res = await api.post<LoginResponse>('/auth/login', data)
      localStorage.setItem('accessToken', res.data.accessToken)
      localStorage.setItem('refreshToken', res.data.refreshToken)
      localStorage.setItem('userId', res.data.userId)
      localStorage.setItem('username', res.data.username)
      set({
        isAuthenticated: true,
        userId: res.data.userId,
        username: res.data.username,
        isLoading: false,
      })
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string; lockedUntil?: string } } }
      const msg = error.response?.data?.error ?? 'Login failed'
      const lockedUntil = error.response?.data?.lockedUntil
      set({
        isLoading: false,
        error: lockedUntil ? `${msg} — locked until ${new Date(lockedUntil).toLocaleTimeString()}` : msg,
      })
      throw err
    }
  },

  register: async (data: RegisterRequest) => {
    set({ isLoading: true, error: null })
    try {
      const res = await api.post<RegisterResponse>('/auth/register', data)
      set({ isLoading: false })
      return res.data
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } }
      const msg = error.response?.data?.error ?? 'Registration failed'
      set({ isLoading: false, error: msg })
      throw err
    }
  },

  logout: async () => {
    try {
      await api.post('/auth/logout')
    } catch {
      // Ignore logout errors
    } finally {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('refreshToken')
      localStorage.removeItem('userId')
      localStorage.removeItem('username')
      set({ isAuthenticated: false, userId: null, username: null })
    }
  },
}))
