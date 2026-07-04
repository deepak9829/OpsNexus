import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Edit, Trash2 } from 'lucide-react'
import { useTenants, useDeactivateTenant } from '@/hooks/useTenants'
import { Table } from '@/components/ui/Table'
import { Badge, planBadge, statusBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { NewTenantModal } from './NewTenantModal'
import { formatDate } from '@/utils/format'
import type { Tenant } from '@/types'

export function TenantsPage() {
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [deactivateId, setDeactivateId] = useState<string | null>(null)

  const { data, isLoading } = useTenants({ page, limit: 20, search: search || undefined })
  const deactivate = useDeactivateTenant()

  const columns = [
    {
      key: 'name',
      header: 'Name',
      sortable: true,
      render: (row: Tenant) => (
        <div>
          <button
            onClick={() => navigate(`/tenants/${row.id}`)}
            className="font-medium text-indigo-600 hover:text-indigo-800 text-left"
          >
            {row.name}
          </button>
          <p className="text-xs text-slate-400">{row.slug}</p>
        </div>
      ),
    },
    {
      key: 'plan',
      header: 'Plan',
      render: (row: Tenant) => (
        <Badge variant={planBadge(row.plan)}>{row.plan}</Badge>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (row: Tenant) => (
        <Badge variant={statusBadge(row.status)}>{row.status}</Badge>
      ),
    },
    {
      key: 'createdAt',
      header: 'Created',
      sortable: true,
      render: (row: Tenant) => (
        <span className="text-slate-600">{formatDate(row.createdAt)}</span>
      ),
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row: Tenant) => (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate(`/tenants/${row.id}`)}
          >
            <Edit className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setDeactivateId(row.id)}
            className="text-red-500 hover:text-red-700 hover:bg-red-50"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ),
    },
  ]

  const handleDeactivate = async () => {
    if (deactivateId) {
      await deactivate.mutateAsync(deactivateId)
      setDeactivateId(null)
    }
  }

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Tenants</h1>
          <p className="text-slate-500 text-sm mt-0.5">
            {data?.meta.total ?? 0} total tenants
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-1.5" />
          New Tenant
        </Button>
      </div>

      {/* Search */}
      <div className="max-w-xs">
        <Input
          placeholder="Search by name or slug..."
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          className="pl-9"
        />
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table
          columns={columns}
          data={data?.data ?? []}
          loading={isLoading}
          emptyMessage="No tenants found. Create one to get started."
        />
      </div>

      {/* Pagination */}
      {data && (
        <Pagination
          page={page}
          totalPages={data.meta.totalPages}
          onPageChange={setPage}
        />
      )}

      {/* Modals */}
      <NewTenantModal open={showCreate} onClose={() => setShowCreate(false)} />
      <ConfirmDialog
        open={!!deactivateId}
        onClose={() => setDeactivateId(null)}
        onConfirm={handleDeactivate}
        title="Deactivate Tenant"
        message="Are you sure you want to deactivate this tenant? Their users will lose access immediately."
        confirmLabel="Deactivate"
        loading={deactivate.isPending}
      />
    </div>
  )
}
