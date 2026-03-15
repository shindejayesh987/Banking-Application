import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { DashboardPage } from '@/pages/DashboardPage'

describe('DashboardPage', () => {
  it('renders the Dashboard heading', () => {
    render(<DashboardPage />)
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders the subtitle', () => {
    render(<DashboardPage />)
    expect(screen.getByText('System health at a glance')).toBeInTheDocument()
  })

  it('renders all 6 system design cards', () => {
    render(<DashboardPage />)
    const cards = ['Redis Cache', 'Kafka', 'Circuit Breaker', 'Rate Limiter', 'DB Replication', 'Saga']
    for (const name of cards) {
      expect(screen.getByText(name)).toBeInTheDocument()
    }
  })

  it('renders 6 cards total', () => {
    render(<DashboardPage />)
    // Each card has "Waiting for backend connection..." text
    const waitingTexts = screen.getAllByText('Waiting for backend connection...')
    expect(waitingTexts).toHaveLength(6)
  })
})
