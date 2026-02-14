import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import App from '../src/App'

// Mock fetch globally
beforeEach(() => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => [],
  } as Response)
})

function renderWithProviders(ui: React.ReactElement, { route = '/' } = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[route]}>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe('App', () => {
  it('renders the nav with Infraplane brand', () => {
    renderWithProviders(<App />)
    expect(screen.getByText('Infraplane')).toBeInTheDocument()
  })

  it('renders nav links', () => {
    renderWithProviders(<App />)
    expect(screen.getByText('Applications')).toBeInTheDocument()
    expect(screen.getByText('Deployments')).toBeInTheDocument()
    expect(screen.getByText('Migration')).toBeInTheDocument()
  })

  it('renders ApplicationList on /', async () => {
    renderWithProviders(<App />, { route: '/' })
    // The heading should be present
    expect(screen.getByText('Applications')).toBeInTheDocument()
  })

  it('renders DeploymentDashboard on /deployments', async () => {
    renderWithProviders(<App />, { route: '/deployments' })
    expect(await screen.findByText('Deployment Dashboard')).toBeInTheDocument()
  })

  it('renders MigrationPlanner on /migrate', async () => {
    renderWithProviders(<App />, { route: '/migrate' })
    expect(await screen.findByText('Migration Planner')).toBeInTheDocument()
  })
})
