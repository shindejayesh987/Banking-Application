import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { PlaceholderPage } from '@/pages/PlaceholderPage'

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="*" element={<PlaceholderPage />} />
      </Routes>
    </MemoryRouter>,
  )
}

describe('PlaceholderPage', () => {
  it('renders "coming soon" message', () => {
    renderAt('/system/redis')
    expect(screen.getByText('This page will be built in a future phase.')).toBeInTheDocument()
  })

  it('derives title from single-segment path', () => {
    renderAt('/accounts')
    expect(screen.getByText('Accounts')).toBeInTheDocument()
  })

  it('derives title from multi-segment path using > separator', () => {
    renderAt('/system/circuit-breaker')
    expect(screen.getByText('System > Circuit-breaker')).toBeInTheDocument()
  })

  it('capitalizes each segment', () => {
    renderAt('/system/kafka')
    expect(screen.getByText('System > Kafka')).toBeInTheDocument()
  })

  it('shows Page as fallback for root path', () => {
    renderAt('/')
    // pathname is '/', split by '/' and filter(Boolean) gives [], so title is ''
    // The component renders title || 'Page'
    expect(screen.getByText('Page')).toBeInTheDocument()
  })
})
