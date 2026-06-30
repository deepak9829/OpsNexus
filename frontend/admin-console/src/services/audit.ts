import { apiClient } from './api'
import type { AuditEvent, PaginatedResponse } from '@/types'

export interface AuditFilter {
  actorId?: string
  actorEmail?: string
  action?: string
  resource?: string
  from?: string
  to?: string
  page?: number
  limit?: number
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapAuditEvent(e: any): AuditEvent {
  return {
    id: e.id ?? e.ID,
    tenantId: e.tenant_id ?? e.tenantId ?? e.TenantID,
    actorId: e.actor_id ?? e.actorId ?? e.ActorID,
    actorEmail: e.actor_email ?? e.actorEmail ?? e.ActorEmail,
    action: e.action ?? e.Action,
    resource: e.resource ?? e.Resource,
    resourceId: e.resource_id ?? e.resourceId ?? e.ResourceID,
    oldValue: e.old_value ?? e.oldValue,
    newValue: e.new_value ?? e.newValue,
    ipAddress: e.ip_address ?? e.ipAddress ?? e.IPAddress,
    userAgent: e.user_agent ?? e.userAgent ?? e.UserAgent,
    timestamp: e.timestamp ?? e.Timestamp,
  }
}

export const auditService = {
  list: async (filter?: AuditFilter): Promise<PaginatedResponse<AuditEvent>> => {
    const { data } = await apiClient.get('/audit-events', { params: filter })
    const total = data.total ?? 0
    const limit = data.limit ?? 25
    return {
      data: (data.data ?? []).map(mapAuditEvent),
      meta: { page: data.page ?? 1, limit, total, totalPages: Math.ceil(total / limit) },
    }
  },
  get: async (id: string): Promise<AuditEvent> => {
    const { data } = await apiClient.get(`/audit-events/${id}`)
    return mapAuditEvent(data.data ?? data)
  },
}
