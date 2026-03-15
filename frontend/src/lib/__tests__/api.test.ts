import { describe, it, expect, beforeEach, vi } from 'vitest'

// We test the interceptor behavior by checking what gets added to request config.
// We inspect the axios instance's request interceptors directly.

describe('api interceptor', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('attaches Bearer token when accessToken is in localStorage', async () => {
    localStorage.setItem('accessToken', 'test-token-123')

    // Re-import api fresh so it uses the current localStorage
    vi.resetModules()
    const { default: api } = await import('@/lib/api')

    // Inspect the request interceptors by calling them manually
    const interceptors = (api.interceptors.request as unknown as {
      handlers: Array<{ fulfilled: (config: unknown) => unknown }>
    }).handlers

    const config = { headers: {} as Record<string, string> }
    const modified = interceptors[0].fulfilled(config) as { headers: Record<string, string> }
    expect(modified.headers.Authorization).toBe('Bearer test-token-123')
  })

  it('does not attach Authorization header when no token', async () => {
    vi.resetModules()
    const { default: api } = await import('@/lib/api')

    const interceptors = (api.interceptors.request as unknown as {
      handlers: Array<{ fulfilled: (config: unknown) => unknown }>
    }).handlers

    const config = { headers: {} as Record<string, string> }
    const modified = interceptors[0].fulfilled(config) as { headers: Record<string, string> }
    expect(modified.headers.Authorization).toBeUndefined()
  })

  it('is an axios instance', async () => {
    vi.resetModules()
    const { default: api } = await import('@/lib/api')
    expect(typeof api.get).toBe('function')
    expect(typeof api.post).toBe('function')
  })
})
