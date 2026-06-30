import { createContext, useCallback, useEffect, useState, type ReactNode } from 'react'
import { authService } from '@/services/auth'
import type { User } from '@/types'
import { FullPageSpinner } from '@/components/ui/Spinner'

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  isAdmin: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

const ADMIN_ROLES = ['admin', 'super_admin', 'platform_admin']

function hasAdminRole(user: User): boolean {
  return user.roles.some((r) => ADMIN_ROLES.includes(r.name.toLowerCase()))
}

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
        if (hasAdminRole(u)) {
          setUser(u)
        } else {
          localStorage.clear()
        }
      })
      .catch(() => {
        localStorage.clear()
      })
      .finally(() => setLoading(false))
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const tokens = await authService.login({ email, password })
    localStorage.setItem('access_token', tokens.accessToken)
    localStorage.setItem('refresh_token', tokens.refreshToken)
    const me = await authService.me()
    if (!hasAdminRole(me)) {
      localStorage.clear()
      throw new Error('ACCESS_DENIED')
    }
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
      localStorage.clear()
      setUser(null)
    }
  }, [])

  if (loading) return <FullPageSpinner />

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isAdmin: !!user && hasAdminRole(user),
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}
