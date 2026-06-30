import { apiClient } from './api'
import type { User, Role, PaginatedResponse } from '@/types'

export interface ListUsersParams {
  page?: number
  limit?: number
  tenantId?: string
  role?: string
  status?: string
  search?: string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapUser(u: any): User {
  return {
    id: u.id ?? u.ID,
    tenantId: u.tenant_id ?? u.tenantId ?? u.TenantID,
    email: u.email ?? u.Email,
    firstName: u.first_name ?? u.firstName ?? u.FirstName,
    lastName: u.last_name ?? u.lastName ?? u.LastName,
    status: u.status ?? u.Status,
    roles: (u.roles ?? u.Roles ?? []).map((r: any) => ({ id: r.id ?? r.ID, name: r.name ?? r.Name, permissions: r.permissions ?? [] })),
    createdAt: u.created_at ?? u.createdAt ?? u.CreatedAt,
    updatedAt: u.updated_at ?? u.updatedAt ?? u.UpdatedAt,
  }
}

export const usersService = {
  list: async (params?: ListUsersParams): Promise<PaginatedResponse<User>> => {
    const { data } = await apiClient.get('/auth/users', { params })
    const total = data.total ?? 0
    const limit = data.limit ?? 20
    return {
      data: (data.data ?? []).map(mapUser),
      meta: { page: data.page ?? 1, limit, total, totalPages: Math.ceil(total / limit) },
    }
  },
  me: async (): Promise<User> => {
    const { data } = await apiClient.get<{ data: unknown }>('/auth/me')
    return mapUser(data.data)
  },
  listRoles: async (): Promise<Role[]> => {
    const { data } = await apiClient.get<{ data: Role[] }>('/auth/roles')
    return data.data
  },
  assignRole: async (userId: string, roleId: string): Promise<void> => {
    await apiClient.post(`/auth/users/${userId}/roles`, { roleId })
  },
  removeRole: async (userId: string, roleId: string): Promise<void> => {
    await apiClient.delete(`/auth/users/${userId}/roles/${roleId}`)
  },
  deactivate: async (userId: string): Promise<void> => {
    await apiClient.patch(`/auth/users/${userId}`, { status: 'inactive' })
  },
}
