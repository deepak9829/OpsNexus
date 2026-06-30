import { useState, useEffect, useCallback, type ReactNode } from 'react'
import { AuthContext } from '@/hooks/useAuth'
import { authService } from '@/services/auth'
import type { User } from '@/types'
import { Spinner } from '@/components/ui/Spinner'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) {
      setLoading(false)
      return
    }
    authService
      .me()
      .then((u) => {
        localStorage.setItem('user_id', u.id)
        setUser(u)
      })
      .catch(() => {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        localStorage.removeItem('tenant_id')
        localStorage.removeItem('user_id')
      })
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const tokens = await authService.login({ email, password })
    localStorage.setItem('access_token', tokens.accessToken)
    localStorage.setItem('refresh_token', tokens.refreshToken)
    const me = await authService.me()
    localStorage.setItem('tenant_id', me.tenantId)
    localStorage.setItem('user_id', me.id)
    setUser(me)
  }, [])

  const logout = useCallback(async () => {
    const refreshToken = localStorage.getItem('refresh_token') ?? ''
    try {
      await authService.logout(refreshToken)
    } catch {
      // best effort
    } finally {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('tenant_id')
      localStorage.removeItem('user_id')
      setUser(null)
    }
  }, [])

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading: false,
        login,
        logout,
        setUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}
