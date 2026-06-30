import { useState } from 'react'
import { Search } from 'lucide-react'
import { useAuditEvents } from '@/hooks/useAudit'
import { Table } from '@/components/ui/Table'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import { AuditEventDetailModal } from './AuditEventDetailModal'
import { formatDateTime } from '@/utils/format'
import type { AuditEvent } from '@/types'

const RESOURCE_TYPES = ['tenant', 'user', 'case', 'role', 'form', 'document', 'workflow']

export function AuditLogPage() {
  const [page, setPage] = useState(1)
  const [actorEmail, setActorEmail] = useState('')
  const [action, setAction] = useState('')
  const [resource, setResource] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [selectedEvent, setSelectedEvent] = useState<AuditEvent | null>(null)

  const { data, isLoading } = useAuditEvents({
    page,
    limit: 25,
    actorEmail: actorEmail || undefined,
    action: action || undefined,
    resource: resource || undefined,
    from: from || undefined,
    to: to || undefined,
  })

  const columns = [
    {
      key: 'timestamp',
      header: 'Timestamp',
      sortable: true,
      render: (e: AuditEvent) => (
        <span className="text-xs text-slate-600 font-mono whitespace-nowrap">{formatDateTime(e.timestamp)}</span>
      ),
    },
    {
      key: 'actorEmail',
      header: 'Actor',
      render: (e: AuditEvent) => (
        <span className="text-sm font-medium text-slate-700">{e.actorEmail}</span>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      render: (e: AuditEvent) => (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-50 text-indigo-700 font-mono">
          {e.action}
        </span>
      ),
    },
    {
      key: 'resource',
      header: 'Resource',
      render: (e: AuditEvent) => (
        <span className="text-sm text-slate-600 capitalize">{e.resource}</span>
      ),
    },
    {
      key: 'resourceId',
      header: 'Resource ID',
      render: (e: AuditEvent) => (
        <span className="text-xs text-slate-400 font-mono truncate block max-w-[120px]">{e.resourceId}</span>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Audit Log</h1>
        <p className="text-slate-500 text-sm mt-0.5">Track all platform activity and changes</p>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-4">
        <p className="text-sm font-medium text-slate-700 mb-3">Filters</p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <Input
            placeholder="Actor email..."
            value={actorEmail}
            onChange={(e) => { setActorEmail(e.target.value); setPage(1) }}
          />
          <Input
            placeholder="Action (e.g. CREATE, DELETE)..."
            value={action}
            onChange={(e) => { setAction(e.target.value); setPage(1) }}
          />
          <div>
            <select
              value={resource}
              onChange={(e) => { setResource(e.target.value); setPage(1) }}
              className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              <option value="">All Resource Types</option>
              {RESOURCE_TYPES.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-500 mb-1">From Date</label>
            <input
              type="date"
              value={from}
              onChange={(e) => { setFrom(e.target.value); setPage(1) }}
              className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-500 mb-1">To Date</label>
            <input
              type="date"
              value={to}
              onChange={(e) => { setTo(e.target.value); setPage(1) }}
              className="block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
          </div>
        </div>
      </div>

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table
          columns={columns}
          data={data?.data ?? []}
          loading={isLoading}
          emptyMessage="No audit events found matching your filters."
          onSort={() => {}}
        />
      </div>

      {data && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-500">
            Showing {data.data.length} of {data.meta.total} events
          </p>
          <Pagination page={page} totalPages={data.meta.totalPages} onPageChange={setPage} />
        </div>
      )}

      <AuditEventDetailModal
        open={!!selectedEvent}
        onClose={() => setSelectedEvent(null)}
        event={selectedEvent}
      />
    </div>
  )
}
