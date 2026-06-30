import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { tenantsService, type CreateTenantPayload } from '@/services/tenants'
import type { TenantSettings } from '@/types'

export function useTenants(params?: { page?: number; limit?: number; search?: string }) {
  return useQuery({
    queryKey: ['tenants', params],
    queryFn: () => tenantsService.list(params),
  })
}

export function useTenant(id: string) {
  return useQuery({
    queryKey: ['tenants', id],
    queryFn: () => tenantsService.get(id),
    enabled: !!id,
  })
}

export function useTenantSettings(id: string) {
  return useQuery({
    queryKey: ['tenants', id, 'settings'],
    queryFn: () => tenantsService.getSettings(id),
    enabled: !!id,
  })
}

export function useTenantMembers(id: string, params?: { page?: number }) {
  return useQuery({
    queryKey: ['tenants', id, 'members', params],
    queryFn: () => tenantsService.listMembers(id, params),
    enabled: !!id,
  })
}

export function useCreateTenant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateTenantPayload) => tenantsService.create(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  })
}

export function useUpdateTenant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<CreateTenantPayload & { status: string }> }) =>
      tenantsService.update(id, payload),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ['tenants'] })
      qc.invalidateQueries({ queryKey: ['tenants', id] })
    },
  })
}

export function useUpdateTenantSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, settings }: { id: string; settings: Partial<TenantSettings> }) =>
      tenantsService.updateSettings(id, settings),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ['tenants', id, 'settings'] })
    },
  })
}

export function useDeactivateTenant() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => tenantsService.deactivate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tenants'] }),
  })
}
