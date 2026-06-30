import { apiClient } from './api'
import type { Tenant, TenantSettings, PaginatedResponse, User } from '@/types'

export interface CreateTenantPayload {
  name: string
  slug: string
  plan: string
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapTenant(t: any): Tenant {
  return {
    id: t.id ?? t.ID,
    name: t.name ?? t.Name,
    slug: t.slug ?? t.Slug,
    plan: t.plan ?? t.Plan,
    status: t.status ?? t.Status,
    settings: t.settings ?? t.Settings ?? {},
    createdAt: t.created_at ?? t.createdAt ?? t.CreatedAt,
    updatedAt: t.updated_at ?? t.updatedAt ?? t.UpdatedAt,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapMember(u: any) {
  return {
    id: u.id ?? u.ID,
    tenantId: u.tenant_id ?? u.tenantId ?? u.TenantID,
    email: u.email ?? u.Email,
    firstName: u.first_name ?? u.firstName ?? u.FirstName,
    lastName: u.last_name ?? u.lastName ?? u.LastName,
    status: u.status ?? u.Status,
    roles: (u.roles ?? u.Roles ?? []).map((r: any) => ({ id: r.id ?? r.ID, name: r.name ?? r.Name, permissions: [] })),
    createdAt: u.created_at ?? u.createdAt ?? u.CreatedAt,
    updatedAt: u.updated_at ?? u.updatedAt ?? u.UpdatedAt,
  }
}

export const tenantsService = {
  list: async (params?: { page?: number; limit?: number; search?: string }): Promise<PaginatedResponse<Tenant>> => {
    const { data } = await apiClient.get('/tenants', { params })
    const total = data.total ?? 0
    const limit = data.limit ?? 20
    return {
      data: (data.data ?? []).map(mapTenant),
      meta: { page: data.page ?? 1, limit, total, totalPages: Math.ceil(total / limit) },
    }
  },
  get: async (id: string): Promise<Tenant> => {
    const { data } = await apiClient.get(`/tenants/${id}`)
    return mapTenant(data.data ?? data)
  },
  create: async (payload: CreateTenantPayload): Promise<Tenant> => {
    const { data } = await apiClient.post(`/tenants`, payload)
    return mapTenant(data.data ?? data)
  },
  update: async (id: string, payload: Partial<CreateTenantPayload & { status: string }>): Promise<Tenant> => {
    const { data } = await apiClient.put(`/tenants/${id}`, payload)
    return mapTenant(data.data ?? data)
  },
  deactivate: async (id: string): Promise<void> => {
    await apiClient.delete(`/tenants/${id}`)
  },
  getSettings: async (id: string): Promise<TenantSettings> => {
    const { data } = await apiClient.get<{ data: TenantSettings }>(`/tenants/${id}/settings`)
    return data.data
  },
  updateSettings: async (id: string, settings: Partial<TenantSettings>): Promise<TenantSettings> => {
    const { data } = await apiClient.put<{ data: TenantSettings }>(`/tenants/${id}/settings`, settings)
    return data.data
  },
  listMembers: async (id: string, params?: { page?: number }): Promise<PaginatedResponse<User>> => {
    const { data } = await apiClient.get(`/tenants/${id}/members`, { params })
    const total = data.total ?? 0
    const limit = data.limit ?? 20
    return {
      data: (data.data ?? []).map(mapMember),
      meta: { page: data.page ?? 1, limit, total, totalPages: Math.ceil(total / limit) },
    }
  },
}
