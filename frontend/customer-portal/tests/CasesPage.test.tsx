import { describe, it, expect, afterEach, afterAll, beforeAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { CasesPage } from '../src/features/cases/CasesPage'
import { AuthContext, type AuthContextValue } from '../src/hooks/useAuth'
import type { Case, PaginatedResponse } from '../src/types'

// ---------- Mock data ----------

const mockCases: Case[] = [
  {
    id: 'case-1',
    tenantId: 'tenant-1',
    caseNumber: 'CASE-0001',
    title: 'Network connectivity issue',
    description: 'Cannot connect to VPN',
    status: 'open',
    priority: 'high',
    reporterId: 'user-1',
    sla: { breached: false },
    tags: ['network'],
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-15T10:00:00Z',
  },
  {
    id: 'case-2',
    tenantId: 'tenant-1',
    caseNumber: 'CASE-0002',
    title: 'Billing discrepancy',
    description: 'Invoice amount is wrong',
    status: 'pending',
    priority: 'medium',
    reporterId: 'user-1',
    sla: { breached: false },
    tags: ['billing'],
    createdAt: '2024-01-14T09:00:00Z',
    updatedAt: '2024-01-14T09:00:00Z',
  },
]

const mockResponse: PaginatedResponse<Case> = {
  data: mockCases,
  meta: { page: 1, limit: 20, total: 2, totalPages: 1 },
}

// ---------- MSW server ----------

const server = setupServer(
  http.get('/api/v1/cases', () => {
    return HttpResponse.json(mockResponse)
  }),
)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// ---------- Helpers ----------

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
}

const mockAuthValue: AuthContextValue = {
  user: {
    id: 'user-1',
    tenantId: 'tenant-1',
    email: 'test@example.com',
    firstName: 'Test',
    lastName: 'User',
    status: 'active',
    roles: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
  isAuthenticated: true,
  isLoading: false,
  login: async () => {},
  logout: async () => {},
  setUser: () => {},
}

function renderCasesPage() {
  const qc = makeQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <AuthContext.Provider value={mockAuthValue}>
          <CasesPage />
        </AuthContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

// ---------- Tests ----------

describe('CasesPage', () => {
  it('renders the page heading', () => {
    renderCasesPage()
    expect(screen.getByText('My Cases')).toBeInTheDocument()
  })

  it('renders New Case button', () => {
    renderCasesPage()
    expect(screen.getByRole('button', { name: /new case/i })).toBeInTheDocument()
  })

  it('renders filter controls', () => {
    renderCasesPage()
    expect(screen.getByPlaceholderText(/search cases/i)).toBeInTheDocument()
    expect(screen.getByText('All Statuses')).toBeInTheDocument()
    expect(screen.getByText('All Priorities')).toBeInTheDocument()
  })

  it('displays cases after loading', async () => {
    renderCasesPage()

    await waitFor(() => {
      expect(screen.getByText('CASE-0001')).toBeInTheDocument()
    })

    expect(screen.getByText('Network connectivity issue')).toBeInTheDocument()
    expect(screen.getByText('CASE-0002')).toBeInTheDocument()
    expect(screen.getByText('Billing discrepancy')).toBeInTheDocument()
  })

  it('displays status badges', async () => {
    renderCasesPage()

    await waitFor(() => {
      expect(screen.getByText('Open')).toBeInTheDocument()
    })

    expect(screen.getByText('Pending')).toBeInTheDocument()
  })

  it('displays priority badges', async () => {
    renderCasesPage()

    await waitFor(() => {
      expect(screen.getByText('High')).toBeInTheDocument()
    })

    expect(screen.getByText('Medium')).toBeInTheDocument()
  })

  it('shows empty state when API returns no cases', async () => {
    server.use(
      http.get('/api/v1/cases', () => {
        return HttpResponse.json({
          data: [],
          meta: { page: 1, limit: 20, total: 0, totalPages: 0 },
        })
      }),
    )

    renderCasesPage()

    await waitFor(() => {
      expect(screen.getByText('No cases found')).toBeInTheDocument()
    })
  })

  it('shows error state when API fails', async () => {
    server.use(
      http.get('/api/v1/cases', () => {
        return new HttpResponse(null, { status: 500 })
      }),
    )

    renderCasesPage()

    await waitFor(() => {
      expect(screen.getByText(/failed to load cases/i)).toBeInTheDocument()
    })
  })
})
