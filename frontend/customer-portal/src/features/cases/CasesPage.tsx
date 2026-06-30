import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Search, ChevronLeft, ChevronRight } from 'lucide-react'
import { useCases } from '@/hooks/useCases'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { CaseStatusBadge, CasePriorityBadge } from '@/components/ui/Badge'
import { PageSpinner } from '@/components/ui/Spinner'
import { EmptyState } from '@/components/ui/EmptyState'
import { NewCaseModal } from './NewCaseModal'
import { formatDate } from '@/utils/format'
import type { CasePriority, CaseStatus } from '@/types'

const STATUS_OPTIONS: { label: string; value: string }[] = [
  { label: 'All Statuses', value: '' },
  { label: 'New', value: 'new' },
  { label: 'Open', value: 'open' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'Pending', value: 'pending' },
  { label: 'Resolved', value: 'resolved' },
  { label: 'Closed', value: 'closed' },
]

const PRIORITY_OPTIONS: { label: string; value: string }[] = [
  { label: 'All Priorities', value: '' },
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
  { label: 'Critical', value: 'critical' },
]

export function CasesPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState('')
  const [priority, setPriority] = useState('')
  const [search, setSearch] = useState('')
  const [isNewCaseOpen, setIsNewCaseOpen] = useState(false)

  const { data, isLoading, isError } = useCases({
    page,
    status: status || undefined,
    priority: priority || undefined,
  })

  const cases = data?.data ?? []
  const meta = data?.meta

  // Client-side search filter
  const filtered = search
    ? cases.filter(
        (c) =>
          c.title.toLowerCase().includes(search.toLowerCase()) ||
          c.caseNumber.toLowerCase().includes(search.toLowerCase()),
      )
    : cases

  const handleFilterChange = () => {
    setPage(1)
  }

  return (
    <div className="space-y-4">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">My Cases</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            {meta ? `${meta.total} case${meta.total !== 1 ? 's' : ''} total` : ''}
          </p>
        </div>
        <Button onClick={() => setIsNewCaseOpen(true)}>
          <Plus className="h-4 w-4" />
          New Case
        </Button>
      </div>

      {/* Filter bar */}
      <Card>
        <div className="px-4 py-3 flex flex-wrap items-center gap-3">
          {/* Search */}
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <input
              type="text"
              placeholder="Search cases..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 pr-3 py-2 w-full rounded-md border border-gray-300 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          {/* Status filter */}
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value)
              handleFilterChange()
            }}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>

          {/* Priority filter */}
          <select
            value={priority}
            onChange={(e) => {
              setPriority(e.target.value)
              handleFilterChange()
            }}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {PRIORITY_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
      </Card>

      {/* Cases table */}
      <Card>
        {isLoading ? (
          <PageSpinner />
        ) : isError ? (
          <div className="py-8 text-center text-sm text-red-600">
            Failed to load cases. Please refresh.
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title="No cases found"
            description="Try adjusting your filters or create your first case."
            action={{ label: 'New Case', onClick: () => setIsNewCaseOpen(true) }}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    {['Case #', 'Title', 'Status', 'Priority', 'Created', 'SLA'].map((h) => (
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
                  {filtered.map((c) => (
                    <tr
                      key={c.id}
                      onClick={() => navigate(`/cases/${c.id}`)}
                      className="cursor-pointer hover:bg-gray-50 transition-colors"
                    >
                      <td className="px-4 py-3 text-sm font-mono font-medium text-blue-600">
                        {c.caseNumber}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900 max-w-xs truncate">
                        {c.title}
                      </td>
                      <td className="px-4 py-3">
                        <CaseStatusBadge status={c.status as CaseStatus} />
                      </td>
                      <td className="px-4 py-3">
                        <CasePriorityBadge priority={c.priority as CasePriority} />
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500">
                        {formatDate(c.createdAt)}
                      </td>
                      <td className="px-4 py-3 text-sm">
                        {c.sla.breached ? (
                          <span className="text-red-600 font-medium">Breached</span>
                        ) : c.sla.dueAt ? (
                          <span className="text-gray-500">{formatDate(c.sla.dueAt)}</span>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {meta && meta.totalPages > 1 && (
              <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200">
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

      <NewCaseModal
        isOpen={isNewCaseOpen}
        onClose={() => setIsNewCaseOpen(false)}
      />
    </div>
  )
}
