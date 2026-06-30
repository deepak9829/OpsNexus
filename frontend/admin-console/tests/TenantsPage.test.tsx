import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TenantsPage } from '@/features/tenants/TenantsPage'

const mockTenants = [
  {
    id: '1',
    name: 'Acme Corp',
    slug: 'acme-corp',
    plan: 'enterprise' as const,
    status: 'active' as const,
    createdAt: '2024-01-15T00:00:00Z',
    updatedAt: '2024-01-15T00:00:00Z',
    settings: {
      maxUsers: 200,
      allowedDomains: [],
      features: {},
      notificationPrefs: { emailEnabled: true, smsEnabled: false, inAppEnabled: true },
    },
  },
  {
    id: '2',
    name: 'Beta LLC',
    slug: 'beta-llc',
    plan: 'pro' as const,
    status: 'inactive' as const,
    createdAt: '2024-02-20T00:00:00Z',
    updatedAt: '2024-02-20T00:00:00Z',
    settings: {
      maxUsers: 50,
      allowedDomains: [],
      features: {},
      notificationPrefs: { emailEnabled: true, smsEnabled: false, inAppEnabled: true },
    },
  },
]

vi.mock('@/hooks/useTenants', () => ({
  useTenants: vi.fn(() => ({
    data: {
      data: mockTenants,
      meta: { page: 1, limit: 20, total: 2, totalPages: 1 },
    },
    isLoading: false,
  })),
  useDeactivateTenant: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
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

describe('TenantsPage', () => {
  it('renders page title', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('Tenants')).toBeInTheDocument()
  })

  it('shows New Tenant button', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('New Tenant')).toBeInTheDocument()
  })

  it('renders tenant list from API data', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('Acme Corp')).toBeInTheDocument()
    expect(screen.getByText('Beta LLC')).toBeInTheDocument()
  })

  it('shows plan badges', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('enterprise')).toBeInTheDocument()
    expect(screen.getByText('pro')).toBeInTheDocument()
  })

  it('shows status badges', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('active')).toBeInTheDocument()
    expect(screen.getByText('inactive')).toBeInTheDocument()
  })

  it('opens create modal when New Tenant is clicked', () => {
    render(<TenantsPage />, { wrapper })
    fireEvent.click(screen.getByText('New Tenant'))
    // Modal title should appear
    expect(screen.getByText('Create New Tenant')).toBeInTheDocument()
  })

  it('shows search input', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByPlaceholderText('Search by name or slug...')).toBeInTheDocument()
  })

  it('shows total count', () => {
    render(<TenantsPage />, { wrapper })
    expect(screen.getByText('2 total tenants')).toBeInTheDocument()
  })
})
