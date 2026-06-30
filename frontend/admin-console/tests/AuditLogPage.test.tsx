import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuditLogPage } from '@/features/audit/AuditLogPage'

const mockAuditEvents = [
  {
    id: 'evt-1',
    tenantId: 'tenant-1',
    actorId: 'user-1',
    actorEmail: 'alice@example.com',
    action: 'CREATE',
    resource: 'tenant',
    resourceId: 'tenant-1',
    ipAddress: '192.168.1.1',
    userAgent: 'Mozilla/5.0',
    timestamp: '2024-06-01T10:00:00Z',
  },
  {
    id: 'evt-2',
    tenantId: 'tenant-1',
    actorId: 'user-2',
    actorEmail: 'bob@example.com',
    action: 'DELETE',
    resource: 'user',
    resourceId: 'user-99',
    oldValue: { status: 'active' },
    newValue: { status: 'inactive' },
    ipAddress: '10.0.0.1',
    userAgent: 'Chrome/100',
    timestamp: '2024-06-02T14:30:00Z',
  },
]

vi.mock('@/hooks/useAudit', () => ({
  useAuditEvents: vi.fn(() => ({
    data: {
      data: mockAuditEvents,
      meta: { page: 1, limit: 25, total: 2, totalPages: 1 },
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

describe('AuditLogPage', () => {
  it('renders page title', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByText('Audit Log')).toBeInTheDocument()
  })

  it('renders filter bar', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByPlaceholderText('Actor email...')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('Action (e.g. CREATE, DELETE)...')).toBeInTheDocument()
  })

  it('renders audit events in table', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
    expect(screen.getByText('bob@example.com')).toBeInTheDocument()
  })

  it('shows action badges', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByText('CREATE')).toBeInTheDocument()
    expect(screen.getByText('DELETE')).toBeInTheDocument()
  })

  it('shows resource types', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByText('tenant')).toBeInTheDocument()
    expect(screen.getByText('user')).toBeInTheDocument()
  })

  it('shows event count', () => {
    render(<AuditLogPage />, { wrapper })
    expect(screen.getByText('Showing 2 of 2 events')).toBeInTheDocument()
  })

  it('resource type dropdown includes expected options', () => {
    render(<AuditLogPage />, { wrapper })
    const select = screen.getByDisplayValue('All Resource Types')
    expect(select).toBeInTheDocument()
  })

  it('updates actor email filter on input', () => {
    render(<AuditLogPage />, { wrapper })
    const input = screen.getByPlaceholderText('Actor email...')
    fireEvent.change(input, { target: { value: 'alice@example.com' } })
    expect((input as HTMLInputElement).value).toBe('alice@example.com')
  })
})
