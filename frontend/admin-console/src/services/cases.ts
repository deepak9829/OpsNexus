import { apiClient } from './api'
import type { Case, PaginatedResponse } from '@/types'

export interface ListCasesParams {
  page?: number
  limit?: number
  tenantId?: string
  status?: string
  priority?: string
  assigneeId?: string
  slaBreached?: boolean
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapCase(c: any): Case {
  return {
    id: c.ID ?? c.id,
    tenantId: c.TenantID ?? c.tenantId ?? c.tenant_id,
    caseNumber: c.CaseNumber ?? c.caseNumber ?? c.case_number,
    title: c.Title ?? c.title,
    description: c.Description ?? c.description,
    status: c.Status ?? c.status,
    priority: c.Priority ?? c.priority,
    assigneeId: c.AssigneeID ?? c.assigneeId ?? c.assignee_id,
    reporterId: c.ReporterID ?? c.reporterId ?? c.reporter_id,
    workflowId: c.WorkflowID ?? c.workflowId ?? c.workflow_id,
    sla: { dueAt: c.SLA?.DueAt ?? c.sla?.dueAt, breached: c.SLA?.Breached ?? c.sla?.breached ?? false },
    tags: c.Tags ?? c.tags ?? [],
    createdAt: c.CreatedAt ?? c.createdAt ?? c.created_at,
    updatedAt: c.UpdatedAt ?? c.updatedAt ?? c.updated_at,
    resolvedAt: c.ResolvedAt ?? c.resolvedAt ?? c.resolved_at,
  }
}

export const casesService = {
  list: async (params?: ListCasesParams): Promise<PaginatedResponse<Case>> => {
    const { data } = await apiClient.get('/cases', { params })
    const total = data.total ?? 0
    const limit = data.limit ?? data.meta?.limit ?? 20
    return {
      data: (data.data ?? []).map(mapCase),
      meta: { page: data.page ?? data.meta?.page ?? 1, limit, total, totalPages: Math.ceil(total / limit) },
    }
  },
  get: async (id: string): Promise<Case> => {
    const { data } = await apiClient.get<{ data: Case }>(`/cases/${id}`)
    return data.data
  },
  bulkUpdate: async (ids: string[], payload: Partial<Pick<Case, 'status' | 'priority' | 'assigneeId'>>): Promise<void> => {
    await apiClient.patch('/cases/bulk', { ids, ...payload })
  },
}
