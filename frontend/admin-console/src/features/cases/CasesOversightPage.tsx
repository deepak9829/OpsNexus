import { useState } from 'react'
import { AlertTriangle } from 'lucide-react'
import { useCases } from '@/hooks/useCases'
import { Table } from '@/components/ui/Table'
import { Badge, statusBadge, priorityBadge } from '@/components/ui/Badge'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import { cn } from '@/utils/cn'
import { formatDate } from '@/utils/format'
import type { Case } from '@/types'

export function CasesOversightPage() {
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [priorityFilter, setPriorityFilter] = useState('')
  const [slaBreached, setSlaBreached] = useState(false)

  const { data, isLoading } = useCases({
    page,
    limit: 20,
    status: statusFilter || undefined,
    priority: priorityFilter || undefined,
    slaBreached: slaBreached || undefined,
  })

  const columns = [
    {
      key: 'caseNumber',
      header: 'Case #',
      render: (c: Case) => (
        <span className="font-mono text-xs font-semibold text-slate-600 bg-slate-100 px-2 py-1 rounded">
          {c.caseNumber}
        </span>
      ),
    },
    {
      key: 'title',
      header: 'Title',
      render: (c: Case) => (
        <div className="max-w-xs">
          <p className="font-medium text-slate-800 truncate">{c.title}</p>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (c: Case) => <Badge variant={statusBadge(c.status)}>{c.status.replace(/_/g, ' ')}</Badge>,
    },
    {
      key: 'priority',
      header: 'Priority',
      render: (c: Case) => <Badge variant={priorityBadge(c.priority)}>{c.priority}</Badge>,
    },
    {
      key: 'sla',
      header: 'SLA',
      render: (c: Case) => (
        <div className="flex items-center gap-1">
          {c.sla.breached ? (
            <>
              <AlertTriangle className="h-4 w-4 text-red-500" />
              <span className="text-xs font-semibold text-red-600">BREACHED</span>
            </>
          ) : c.sla.dueAt ? (
            <span className="text-xs text-slate-500">{formatDate(c.sla.dueAt)}</span>
          ) : (
            <span className="text-xs text-slate-400">No SLA</span>
          )}
        </div>
      ),
    },
    {
      key: 'createdAt',
      header: 'Created',
      render: (c: Case) => <span className="text-xs text-slate-500">{formatDate(c.createdAt)}</span>,
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Cases Oversight</h1>
          <p className="text-slate-500 text-sm mt-0.5">All cases across all tenants</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap items-center">
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1) }}
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          <option value="">All Statuses</option>
          <option value="new">New</option>
          <option value="open">Open</option>
          <option value="in_progress">In Progress</option>
          <option value="pending">Pending</option>
          <option value="resolved">Resolved</option>
          <option value="closed">Closed</option>
        </select>
        <select
          value={priorityFilter}
          onChange={(e) => { setPriorityFilter(e.target.value); setPage(1) }}
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          <option value="">All Priorities</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
          <option value="critical">Critical</option>
        </select>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={slaBreached}
            onChange={(e) => { setSlaBreached(e.target.checked); setPage(1) }}
            className="rounded border-slate-300 text-red-600 focus:ring-red-500"
          />
          <span className="text-sm text-slate-700 font-medium flex items-center gap-1">
            <AlertTriangle className="h-4 w-4 text-red-500" />
            SLA Breached Only
          </span>
        </label>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table
          columns={columns}
          data={data?.data ?? []}
          loading={isLoading}
          emptyMessage="No cases found matching your filters."
        />
      </div>

      {data && (
        <Pagination page={page} totalPages={data.meta.totalPages} onPageChange={setPage} />
      )}
    </div>
  )
}
