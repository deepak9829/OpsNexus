import { useNavigate } from 'react-router-dom'
import { FolderOpen, Clock, CheckCircle, AlertCircle } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { useCases } from '@/hooks/useCases'
import { useNotifications } from '@/hooks/useNotifications'
import { Card, CardHeader, CardBody } from '@/components/ui/Card'
import { CaseStatusBadge, CasePriorityBadge } from '@/components/ui/Badge'
import { Spinner } from '@/components/ui/Spinner'
import { formatDate, formatRelative } from '@/utils/format'
import type { CaseStatus } from '@/types'

function StatCard({
  label,
  value,
  icon,
  color,
  loading,
}: {
  label: string
  value: number | string
  icon: React.ReactNode
  color: string
  loading?: boolean
}) {
  return (
    <Card>
      <CardBody className="flex items-center gap-4">
        <div className={`flex h-12 w-12 items-center justify-center rounded-lg ${color}`}>
          {icon}
        </div>
        <div>
          <p className="text-sm text-gray-500">{label}</p>
          {loading ? (
            <Spinner size="sm" className="mt-1" />
          ) : (
            <p className="text-2xl font-bold text-gray-900">{value}</p>
          )}
        </div>
      </CardBody>
    </Card>
  )
}

function SkeletonRow() {
  return (
    <tr className="animate-pulse">
      {[1, 2, 3, 4, 5].map((i) => (
        <td key={i} className="px-4 py-3">
          <div className="h-4 bg-gray-200 rounded w-3/4" />
        </td>
      ))}
    </tr>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const navigate = useNavigate()

  const { data: allCases, isLoading: casesLoading } = useCases({ page: 1, limit: 5 } as Parameters<typeof useCases>[0])
  const { data: openCases } = useCases({ status: 'open' } as Parameters<typeof useCases>[0])
  const { data: pendingCases } = useCases({ status: 'pending' } as Parameters<typeof useCases>[0])
  const { data: resolvedCases } = useCases({ status: 'resolved' } as Parameters<typeof useCases>[0])
  const { data: notifications, isLoading: notifsLoading } = useNotifications(1)

  const recentCases = allCases?.data.slice(0, 5) ?? []
  const unreadNotifications = (notifications?.data ?? []).filter((n) => !n.read).slice(0, 3)

  const notifTypeColor: Record<string, string> = {
    info: 'bg-blue-100 text-blue-700',
    warning: 'bg-yellow-100 text-yellow-700',
    error: 'bg-red-100 text-red-700',
    success: 'bg-green-100 text-green-700',
  }

  return (
    <div className="space-y-6">
      {/* Welcome */}
      <div>
        <h1 className="text-2xl font-bold text-gray-900">
          Welcome back, {user?.firstName}!
        </h1>
        <p className="text-sm text-gray-500 mt-1">
          Here&apos;s a summary of your support activity.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Cases"
          value={allCases?.meta.total ?? 0}
          icon={<FolderOpen className="h-6 w-6 text-blue-600" />}
          color="bg-blue-50"
          loading={casesLoading}
        />
        <StatCard
          label="Open Cases"
          value={openCases?.meta.total ?? 0}
          icon={<AlertCircle className="h-6 w-6 text-orange-600" />}
          color="bg-orange-50"
          loading={casesLoading}
        />
        <StatCard
          label="Pending"
          value={pendingCases?.meta.total ?? 0}
          icon={<Clock className="h-6 w-6 text-yellow-600" />}
          color="bg-yellow-50"
          loading={casesLoading}
        />
        <StatCard
          label="Resolved This Month"
          value={resolvedCases?.meta.total ?? 0}
          icon={<CheckCircle className="h-6 w-6 text-green-600" />}
          color="bg-green-50"
          loading={casesLoading}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Cases */}
        <div className="lg:col-span-2">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold text-gray-900">Recent Cases</h2>
                <button
                  onClick={() => navigate('/cases')}
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  View all
                </button>
              </div>
            </CardHeader>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    {['Case #', 'Title', 'Status', 'Priority', 'Created'].map((h) => (
                      <th
                        key={h}
                        className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {casesLoading
                    ? Array.from({ length: 3 }).map((_, i) => <SkeletonRow key={i} />)
                    : recentCases.map((c) => (
                        <tr
                          key={c.id}
                          onClick={() => navigate(`/cases/${c.id}`)}
                          className="cursor-pointer hover:bg-gray-50 transition-colors"
                        >
                          <td className="px-4 py-3 text-sm font-mono text-blue-600">
                            {c.caseNumber}
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-900 max-w-xs truncate">
                            {c.title}
                          </td>
                          <td className="px-4 py-3">
                            <CaseStatusBadge status={c.status as CaseStatus} />
                          </td>
                          <td className="px-4 py-3">
                            <CasePriorityBadge priority={c.priority} />
                          </td>
                          <td className="px-4 py-3 text-sm text-gray-500">
                            {formatDate(c.createdAt)}
                          </td>
                        </tr>
                      ))}
                  {!casesLoading && recentCases.length === 0 && (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-sm text-gray-500">
                        No cases yet.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </Card>
        </div>

        {/* Recent Notifications */}
        <div>
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold text-gray-900">Notifications</h2>
                <button
                  onClick={() => navigate('/notifications')}
                  className="text-sm text-blue-600 hover:text-blue-800"
                >
                  View all
                </button>
              </div>
            </CardHeader>
            <CardBody className="space-y-3 px-4 py-3">
              {notifsLoading ? (
                <div className="space-y-3">
                  {[1, 2, 3].map((i) => (
                    <div key={i} className="animate-pulse space-y-1">
                      <div className="h-4 bg-gray-200 rounded w-3/4" />
                      <div className="h-3 bg-gray-100 rounded w-1/2" />
                    </div>
                  ))}
                </div>
              ) : unreadNotifications.length === 0 ? (
                <p className="text-sm text-gray-500 text-center py-4">All caught up!</p>
              ) : (
                unreadNotifications.map((n) => (
                  <div
                    key={n.id}
                    className="flex items-start gap-3 rounded-md p-2 hover:bg-gray-50"
                  >
                    <span
                      className={`mt-0.5 inline-flex h-5 w-5 flex-shrink-0 items-center justify-center rounded-full text-xs font-bold ${notifTypeColor[n.type] ?? ''}`}
                    >
                      {n.type.charAt(0).toUpperCase()}
                    </span>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium text-gray-900 truncate">{n.title}</p>
                      <p className="text-xs text-gray-500">{formatRelative(n.createdAt)}</p>
                    </div>
                  </div>
                ))
              )}
            </CardBody>
          </Card>
        </div>
      </div>
    </div>
  )
}
