export interface User {
  id: string
  tenantId: string
  email: string
  firstName: string
  lastName: string
  status: 'active' | 'inactive' | 'suspended'
  roles: Role[]
  createdAt: string
  updatedAt: string
}

export interface Role {
  id: string
  name: string
  permissions: Permission[]
}

export interface Permission {
  id: string
  resource: string
  action: string
}

export interface TokenResponse {
  accessToken: string
  refreshToken: string
  expiresIn: number
  tokenType: string
}

export type CaseStatus = 'new' | 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed'
export type CasePriority = 'low' | 'medium' | 'high' | 'critical'

export interface Case {
  id: string
  tenantId: string
  caseNumber: string
  title: string
  description: string
  status: CaseStatus
  priority: CasePriority
  assigneeId?: string
  reporterId: string
  workflowId?: string
  sla: { dueAt?: string; breached: boolean }
  tags: string[]
  createdAt: string
  updatedAt: string
  resolvedAt?: string
}

export interface Task {
  id: string
  caseId: string
  title: string
  description: string
  status: 'todo' | 'in_progress' | 'done' | 'blocked'
  assigneeId?: string
  dueAt?: string
  completedAt?: string
  createdAt: string
}

export interface Comment {
  id: string
  caseId: string
  authorId: string
  body: string
  createdAt: string
}

export interface Notification {
  id: string
  tenantId: string
  userId: string
  type: 'info' | 'warning' | 'error' | 'success'
  title: string
  body: string
  channel: 'in_app' | 'email' | 'sms'
  read: boolean
  readAt?: string
  createdAt: string
}

export interface FormTemplate {
  id: string
  tenantId: string
  name: string
  description: string
  version: number
  fields: FormField[]
  status: 'draft' | 'published' | 'archived'
  createdAt: string
}

export interface FormField {
  name: string
  type: 'text' | 'email' | 'number' | 'date' | 'select' | 'multiselect' | 'file' | 'textarea'
  label: string
  required: boolean
  placeholder?: string
  options?: string[]
}

export interface Document {
  id: string
  tenantId: string
  filename: string
  mimeType: string
  sizeBytes: number
  uploadedBy: string
  caseId?: string
  versionCount: number
  createdAt: string
}

export interface ApiError {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

export interface PaginatedResponse<T> {
  data: T[]
  meta: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}
