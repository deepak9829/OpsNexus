import { useState } from 'react'
import { UserCheck, UserX } from 'lucide-react'
import { useUsers, useDeactivateUser } from '@/hooks/useUsers'
import { Table } from '@/components/ui/Table'
import { Badge, statusBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Pagination } from '@/components/ui/Pagination'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { AssignRoleModal } from './AssignRoleModal'
import { formatDate } from '@/utils/format'
import type { User } from '@/types'

export function UsersPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [deactivateUser, setDeactivateUser] = useState<User | null>(null)

  const { data, isLoading, error } = useUsers({
    page,
    limit: 20,
    status: statusFilter || undefined,
    search: search || undefined,
  })
  const deactivate = useDeactivateUser()

  const columns = [
    {
      key: 'email',
      header: 'Email',
      sortable: true,
      render: (u: User) => (
        <div>
          <p className="font-medium text-slate-800">{u.email}</p>
        </div>
      ),
    },
    {
      key: 'name',
      header: 'Name',
      render: (u: User) => `${u.firstName} ${u.lastName}`,
    },
    {
      key: 'roles',
      header: 'Roles',
      render: (u: User) => (
        <div className="flex flex-wrap gap-1">
          {u.roles.length > 0 ? (
            u.roles.map((r) => (
              <Badge key={r.id} variant="indigo">{r.name}</Badge>
            ))
          ) : (
            <span className="text-xs text-slate-400 italic">No roles</span>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (u: User) => <Badge variant={statusBadge(u.status)}>{u.status}</Badge>,
    },
    {
      key: 'createdAt',
      header: 'Created',
      render: (u: User) => <span className="text-slate-500 text-xs">{formatDate(u.createdAt)}</span>,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (u: User) => (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setSelectedUser(u)}
            title="Assign Roles"
          >
            <UserCheck className="h-4 w-4 text-indigo-500" />
          </Button>
          {u.status === 'active' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setDeactivateUser(u)}
              title="Deactivate User"
              className="text-red-500 hover:text-red-700 hover:bg-red-50"
            >
              <UserX className="h-4 w-4" />
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Users</h1>
          <p className="text-slate-500 text-sm mt-0.5">{data?.meta.total ?? 0} total users</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="w-64">
          <Input
            placeholder="Search by email..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          />
        </div>
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1) }}
          className="rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
        >
          <option value="">All Statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
          <option value="suspended">Suspended</option>
        </select>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
          Failed to load users. Make sure you are logged in as an admin account.
        </div>
      )}

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table
          columns={columns}
          data={data?.data ?? []}
          loading={isLoading}
          emptyMessage="No users found matching your filters."
        />
      </div>

      {data && (
        <Pagination page={page} totalPages={data.meta.totalPages} onPageChange={setPage} />
      )}

      {selectedUser && (
        <AssignRoleModal
          open={!!selectedUser}
          onClose={() => setSelectedUser(null)}
          user={selectedUser}
        />
      )}

      <ConfirmDialog
        open={!!deactivateUser}
        onClose={() => setDeactivateUser(null)}
        onConfirm={async () => {
          if (deactivateUser) {
            await deactivate.mutateAsync(deactivateUser.id)
            setDeactivateUser(null)
          }
        }}
        title="Deactivate User"
        message={`Are you sure you want to deactivate ${deactivateUser?.email}? They will lose access immediately.`}
        confirmLabel="Deactivate"
        loading={deactivate.isPending}
      />
    </div>
  )
}
