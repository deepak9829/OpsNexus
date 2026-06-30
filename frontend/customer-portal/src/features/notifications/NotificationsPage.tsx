import { useState } from 'react'
import { Bell, CheckCheck, X, ChevronLeft, ChevronRight } from 'lucide-react'
import { clsx } from 'clsx'
import { useNotifications, useMarkRead, useMarkAllRead, useDismissNotification } from '@/hooks/useNotifications'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Badge } from '@/components/ui/Badge'
import { PageSpinner } from '@/components/ui/Spinner'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatRelative } from '@/utils/format'
import type { Notification } from '@/types'

const TYPE_VARIANT: Record<Notification['type'], 'info' | 'warning' | 'danger' | 'success'> = {
  info: 'info',
  warning: 'warning',
  error: 'danger',
  success: 'success',
}

function NotificationItem({ notification }: { notification: Notification }) {
  const markRead = useMarkRead()
  const dismiss = useDismissNotification()

  return (
    <div
      className={clsx(
        'flex items-start gap-4 px-5 py-4 transition-colors',
        !notification.read && 'bg-blue-50/50',
        'hover:bg-gray-50',
      )}
    >
      {/* Unread dot */}
      <div className="flex-shrink-0 mt-1">
        {!notification.read ? (
          <span className="block h-2.5 w-2.5 rounded-full bg-blue-500" aria-label="Unread" />
        ) : (
          <span className="block h-2.5 w-2.5 rounded-full bg-transparent" />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2 flex-wrap">
            <Badge variant={TYPE_VARIANT[notification.type]}>
              {notification.type.charAt(0).toUpperCase() + notification.type.slice(1)}
            </Badge>
            <span className="text-sm font-semibold text-gray-900">{notification.title}</span>
          </div>
          <span className="text-xs text-gray-400 flex-shrink-0 mt-0.5">
            {formatRelative(notification.createdAt)}
          </span>
        </div>
        <p className="mt-1 text-sm text-gray-600">{notification.body}</p>
        <div className="mt-2 flex items-center gap-2">
          {!notification.read && (
            <button
              onClick={() => markRead.mutate(notification.id)}
              className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
            >
              Mark as read
            </button>
          )}
          <span className="text-xs text-gray-300">{notification.channel.replace('_', ' ')}</span>
        </div>
      </div>

      {/* Dismiss */}
      <button
        onClick={() => dismiss.mutate(notification.id)}
        className="flex-shrink-0 p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-200 transition-colors"
        aria-label="Dismiss notification"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  )
}

export function NotificationsPage() {
  const [page, setPage] = useState(1)
  const { data, isLoading, isError } = useNotifications(page)
  const markAllRead = useMarkAllRead()

  const notifications = data?.data ?? []
  const meta = data?.meta
  const unreadCount = notifications.filter((n) => !n.read).length

  return (
    <div className="space-y-4 max-w-3xl">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Notifications</h1>
          {unreadCount > 0 && (
            <p className="text-sm text-gray-500 mt-0.5">
              {unreadCount} unread notification{unreadCount !== 1 ? 's' : ''}
            </p>
          )}
        </div>
        {unreadCount > 0 && (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => markAllRead.mutate()}
            loading={markAllRead.isPending}
          >
            <CheckCheck className="h-4 w-4" />
            Mark all read
          </Button>
        )}
      </div>

      {/* List */}
      <Card>
        {isLoading ? (
          <PageSpinner />
        ) : isError ? (
          <div className="py-8 text-center text-sm text-red-600">
            Failed to load notifications.
          </div>
        ) : notifications.length === 0 ? (
          <EmptyState
            icon={<Bell className="h-8 w-8" />}
            title="No notifications"
            description="You're all caught up! New notifications will appear here."
          />
        ) : (
          <>
            <div className="divide-y divide-gray-100">
              {notifications.map((n) => (
                <NotificationItem key={n.id} notification={n} />
              ))}
            </div>

            {/* Pagination */}
            {meta && meta.totalPages > 1 && (
              <div className="flex items-center justify-between px-5 py-3 border-t border-gray-200">
                <p className="text-sm text-gray-500">
                  Page {meta.page} of {meta.totalPages} ({meta.total} total)
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setPage((p) => p + 1)}
                    disabled={page >= meta.totalPages}
                  >
                    Next
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  )
}
