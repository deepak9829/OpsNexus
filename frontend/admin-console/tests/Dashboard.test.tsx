import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { DashboardPage } from '@/features/dashboard/DashboardPage'

// Mock the hooks
vi.mock('@/hooks/useTenants', () => ({
  useTenants: vi.fn(() => ({
    data: {
      data: [
        { id: '1', name: 'Acme Corp', slug: 'acme', plan: 'enterprise', status: 'active', createdAt: '2024-01-01T00:00:00Z', updatedAt: '2024-01-01T00:00:00Z', settings: { maxUsers: 100, allowedDomains: [], features: {}, notificationPrefs: { emailEnabled: true, smsEnabled: false, inAppEnabled: true } } },
        { id: '2', name: 'Beta LLC', slug: 'beta', plan: 'pro', status: 'active', createdAt: '2024-02-01T00:00:00Z', updatedAt: '2024-02-01T00:00:00Z', settings: { maxUsers: 50, allowedDomains: [], features: {}, notificationPrefs: { emailEnabled: true, smsEnabled: false, inAppEnabled: true } } },
      ],
      meta: { page: 1, limit: 100, total: 2, totalPages: 1 },
    },
    isLoading: false,
  })),
}))

vi.mock('@/hooks/useCases', () => ({
  useCases: vi.fn(() => ({
    data: {
      data: [],
      meta: { page: 1, limit: 100, total: 5, totalPages: 1 },
    },
    isLoading: false,
  })),
}))

vi.mock('@/hooks/useAudit', () => ({
  useAuditEvents: vi.fn(() => ({
    data: {
      data: [
        {
          id: 'a1',
          tenantId: '1',
          actorId: 'u1',
          actorEmail: 'admin@example.com',
          action: 'CREATE',
          resource: 'tenant',
          resourceId: 't1',
          ipAddress: '1.2.3.4',
          userAgent: 'Mozilla',
          timestamp: '2024-06-01T10:00:00Z',
        },
      ],
      meta: { page: 1, limit: 10, total: 1, totalPages: 1 },
    },
    isLoading: false,
  })),
}))

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  )
}

describe('DashboardPage', () => {
  it('renders the page title', () => {
    render(<DashboardPage />, { wrapper })
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('renders stat cards', () => {
    render(<DashboardPage />, { wrapper })
    expect(screen.getByText('Total Tenants')).toBeInTheDocument()
    expect(screen.getByText('Active Tenants')).toBeInTheDocument()
    expect(screen.getByText('Total Cases')).toBeInTheDocument()
    expect(screen.getByText('Open Cases')).toBeInTheDocument()
  })

  it('shows tenant count from data', () => {
    render(<DashboardPage />, { wrapper })
    // 2 total tenants
    expect(screen.getAllByText('2').length).toBeGreaterThan(0)
  })

  it('renders tenant health table', () => {
    render(<DashboardPage />, { wrapper })
    expect(screen.getByText('Tenant Health')).toBeInTheDocument()
    expect(screen.getByText('Acme Corp')).toBeInTheDocument()
    expect(screen.getByText('Beta LLC')).toBeInTheDocument()
  })

  it('renders recent activity section', () => {
    render(<DashboardPage />, { wrapper })
    expect(screen.getByText('Recent Activity')).toBeInTheDocument()
    expect(screen.getByText('admin@example.com')).toBeInTheDocument()
  })

  it('renders quick actions', () => {
    render(<DashboardPage />, { wrapper })
    expect(screen.getByText('Quick Actions')).toBeInTheDocument()
    expect(screen.getByText('Create Tenant')).toBeInTheDocument()
  })
})
