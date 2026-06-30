import { useQuery } from '@tanstack/react-query'
import { auditService, type AuditFilter } from '@/services/audit'

export function useAuditEvents(filter?: AuditFilter) {
  return useQuery({
    queryKey: ['audit-events', filter],
    queryFn: () => auditService.list(filter),
  })
}

export function useAuditEvent(id: string) {
  return useQuery({
    queryKey: ['audit-events', id],
    queryFn: () => auditService.get(id),
    enabled: !!id,
  })
}
