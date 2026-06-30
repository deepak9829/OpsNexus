import { apiClient } from './api'
import type { Notification, PaginatedResponse } from '@/types'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapNotification(n: any): Notification {
  return {
    id: n.ID ?? n.id,
    tenantId: n.TenantID ?? n.tenantId ?? n.tenant_id,
    userId: n.UserID ?? n.userId ?? n.user_id,
    type: n.Type ?? n.type,
    title: n.Title ?? n.title,
    body: n.Body ?? n.body,
    channel: n.Channel ?? n.channel,
    read: n.Read ?? n.read ?? false,
    readAt: n.ReadAt ?? n.readAt ?? n.read_at,
    createdAt: n.CreatedAt ?? n.createdAt ?? n.created_at,
  }
}

export const notificationsService = {
  list: async (params?: { page?: number; limit?: number }): Promise<PaginatedResponse<Notification>> => {
    const { data } = await apiClient.get('/notifications', { params })
    const total = data.total ?? 0
    const limit = data.limit ?? 20
    return {
      data: (data.data ?? []).map(mapNotification),
      meta: {
        page: data.page ?? 1,
        limit,
        total,
        totalPages: Math.ceil(total / limit),
      },
    }
  },
  markRead: async (id: string): Promise<void> => {
    await apiClient.put(`/notifications/${id}/read`)
  },
  markAllRead: async (): Promise<void> => {
    await apiClient.put('/notifications/read-all')
  },
  dismiss: async (id: string): Promise<void> => {
    await apiClient.delete(`/notifications/${id}`)
  },
}
