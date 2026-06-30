import { apiClient } from './api'
import type { User, TokenResponse } from '@/types'

export interface LoginPayload { email: string; password: string }
export interface RegisterPayload {
  email: string
  password: string
  firstName: string
  lastName: string
  tenantId: string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapToken(d: any): TokenResponse {
  return {
    accessToken: d.access_token,
    refreshToken: d.refresh_token,
    expiresIn: d.expires_in,
    tokenType: d.token_type,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapUser(d: any): User {
  return {
    id: d.id,
    tenantId: d.tenant_id,
    email: d.email,
    firstName: d.first_name,
    lastName: d.last_name,
    status: d.status,
    roles: d.roles ?? [],
    createdAt: d.created_at,
    updatedAt: d.updated_at,
  }
}

export const authService = {
  login: async (payload: LoginPayload): Promise<TokenResponse> => {
    const { data } = await apiClient.post<{ data: unknown }>('/auth/login', payload)
    return mapToken(data.data)
  },
  register: async (payload: RegisterPayload): Promise<User> => {
    const { data } = await apiClient.post<{ data: unknown }>('/auth/register', payload)
    return mapUser(data.data)
  },
  logout: async (refreshToken: string): Promise<void> => {
    await apiClient.post('/auth/logout', { refreshToken })
  },
  me: async (): Promise<User> => {
    const { data } = await apiClient.get<{ data: unknown }>('/auth/me')
    return mapUser(data.data)
  },
  refresh: async (refreshToken: string): Promise<TokenResponse> => {
    const { data } = await apiClient.post<{ data: unknown }>('/auth/refresh', { refreshToken })
    return mapToken(data.data)
  },
}
