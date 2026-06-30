export type {
  User,
  Role,
  Permission,
  TokenResponse,
  Case,
  CaseStatus,
  CasePriority,
  Task,
  Comment,
  Notification,
  FormTemplate,
  FormField,
  Document,
  ApiError,
  PaginatedResponse,
} from './shared'

export interface Tenant {
  id: string
  name: string
  slug: string
  plan: 'free' | 'pro' | 'enterprise'
  status: 'active' | 'inactive' | 'suspended'
  settings: TenantSettings
  createdAt: string
  updatedAt: string
}

export interface TenantSettings {
  maxUsers: number
  allowedDomains: string[]
  features: Record<string, boolean>
  notificationPrefs: {
    emailEnabled: boolean
    smsEnabled: boolean
    inAppEnabled: boolean
  }
}

export interface Organization {
  id: string
  tenantId: string
  name: string
  type: string
  parentId?: string
  metadata: Record<string, unknown>
  createdAt: string
}

export interface AuditEvent {
  id: string
  tenantId: string
  actorId: string
  actorEmail: string
  action: string
  resource: string
  resourceId: string
  oldValue?: Record<string, unknown>
  newValue?: Record<string, unknown>
  ipAddress: string
  userAgent: string
  timestamp: string
}

export interface AdminUser {
  id: string
  tenantId: string
  email: string
  firstName: string
  lastName: string
  status: 'active' | 'inactive' | 'suspended'
  roles: import('./shared').Role[]
  createdAt: string
  updatedAt: string
  organizationId?: string
  displayName?: string
  lastLoginAt?: string
}
