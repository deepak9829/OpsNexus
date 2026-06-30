import { apiClient } from './api'
import type { Case, Task, Comment, PaginatedResponse } from '@/types'

export interface CreateCasePayload {
  title: string
  description: string
  priority: string
  tags?: string[]
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
    sla: {
      dueAt: c.SLA?.DueAt ?? c.sla?.dueAt ?? c.sla?.due_at,
      breached: c.SLA?.Breached ?? c.sla?.breached ?? false,
    },
    tags: c.Tags ?? c.tags ?? [],
    createdAt: c.CreatedAt ?? c.createdAt ?? c.created_at,
    updatedAt: c.UpdatedAt ?? c.updatedAt ?? c.updated_at,
    resolvedAt: c.ResolvedAt ?? c.resolvedAt ?? c.resolved_at,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapPaginated(raw: any, mapItem: (item: any) => any): PaginatedResponse<any> {
  const total = raw.total ?? 0
  const limit = raw.limit ?? raw.meta?.limit ?? 10
  return {
    data: (raw.data ?? []).map(mapItem),
    meta: {
      page: raw.page ?? raw.meta?.page ?? 1,
      limit,
      total,
      totalPages: Math.ceil(total / limit),
    },
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapTask(t: any): Task {
  return {
    id: t.ID ?? t.id,
    caseId: t.CaseID ?? t.caseId ?? t.case_id,
    title: t.Title ?? t.title,
    description: t.Description ?? t.description,
    status: t.Status ?? t.status,
    assigneeId: t.AssigneeID ?? t.assigneeId ?? t.assignee_id,
    dueAt: t.DueAt ?? t.dueAt ?? t.due_at,
    completedAt: t.CompletedAt ?? t.completedAt ?? t.completed_at,
    createdAt: t.CreatedAt ?? t.createdAt ?? t.created_at,
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapComment(c: any): Comment {
  return {
    id: c.ID ?? c.id,
    caseId: c.CaseID ?? c.caseId ?? c.case_id,
    authorId: c.AuthorID ?? c.authorId ?? c.author_id,
    body: c.Body ?? c.body,
    createdAt: c.CreatedAt ?? c.createdAt ?? c.created_at,
  }
}

export const casesService = {
  list: async (params?: {
    page?: number
    limit?: number
    status?: string
    priority?: string
  }): Promise<PaginatedResponse<Case>> => {
    const { data } = await apiClient.get('/cases', { params })
    return mapPaginated(data, mapCase)
  },
  get: async (id: string): Promise<Case> => {
    const { data } = await apiClient.get(`/cases/${id}`)
    return mapCase(data.data ?? data)
  },
  create: async (payload: CreateCasePayload): Promise<Case> => {
    const { data } = await apiClient.post('/cases', payload)
    return mapCase(data.data ?? data)
  },
  update: async (id: string, payload: Partial<CreateCasePayload>): Promise<Case> => {
    const { data } = await apiClient.put(`/cases/${id}`, payload)
    return mapCase(data.data ?? data)
  },
  transition: async (id: string, toStatus: string, reason?: string): Promise<Case> => {
    const { data } = await apiClient.post(`/cases/${id}/transitions`, { toStatus, reason })
    return mapCase(data.data ?? data)
  },
  listTasks: async (caseId: string): Promise<Task[]> => {
    const { data } = await apiClient.get(`/cases/${caseId}/tasks`)
    const list = data.data ?? data ?? []
    return list.map(mapTask)
  },
  addComment: async (caseId: string, body: string): Promise<Comment> => {
    const { data } = await apiClient.post(`/cases/${caseId}/comments`, { body })
    return mapComment(data.data ?? data)
  },
  listComments: async (caseId: string): Promise<Comment[]> => {
    const { data } = await apiClient.get(`/cases/${caseId}/comments`)
    const list = data.data ?? data ?? []
    return list.map(mapComment)
  },
}
