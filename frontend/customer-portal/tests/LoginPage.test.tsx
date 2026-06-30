import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { LoginPage } from '../src/features/auth/LoginPage'
import { AuthContext, type AuthContextValue } from '../src/hooks/useAuth'

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } })
}

function renderLoginPage(authValue: Partial<AuthContextValue> = {}) {
  const login = vi.fn()
  const contextValue: AuthContextValue = {
    user: null,
    isAuthenticated: false,
    isLoading: false,
    login,
    logout: vi.fn(),
    setUser: vi.fn(),
    ...authValue,
  }

  const qc = makeQueryClient()

  render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <AuthContext.Provider value={contextValue}>
          <LoginPage />
        </AuthContext.Provider>
      </MemoryRouter>
    </QueryClientProvider>,
  )

  return { login }
}

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders email and password fields', () => {
    renderLoginPage()
    expect(screen.getByLabelText(/email address/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
  })

  it('renders the sign in button', () => {
    renderLoginPage()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('renders the OpsNexus heading', () => {
    renderLoginPage()
    expect(screen.getByText('OpsNexus')).toBeInTheDocument()
  })

  it('shows email validation error on submit with empty email', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText(/valid email required/i)).toBeInTheDocument()
    })
  })

  it('shows password validation error when password is too short', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByLabelText(/email address/i), 'test@example.com')
    await user.type(screen.getByLabelText(/password/i), 'abc')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText(/at least 6 characters/i)).toBeInTheDocument()
    })
  })

  it('calls login with correct credentials on valid submit', async () => {
    const user = userEvent.setup()
    const login = vi.fn().mockResolvedValue(undefined)
    renderLoginPage({ login })

    await user.type(screen.getByLabelText(/email address/i), 'test@example.com')
    await user.type(screen.getByLabelText(/password/i), 'password123')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(login).toHaveBeenCalledWith('test@example.com', 'password123')
    })
  })

  it('shows server error message when login fails', async () => {
    const user = userEvent.setup()
    const login = vi.fn().mockRejectedValue({
      response: { data: { error: { message: 'Invalid credentials' } } },
    })
    renderLoginPage({ login })

    await user.type(screen.getByLabelText(/email address/i), 'test@example.com')
    await user.type(screen.getByLabelText(/password/i), 'wrongpass')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => {
      expect(screen.getByText('Invalid credentials')).toBeInTheDocument()
    })
  })

  it('shows forgot password link', () => {
    renderLoginPage()
    expect(screen.getByText(/forgot your password/i)).toBeInTheDocument()
  })
})
