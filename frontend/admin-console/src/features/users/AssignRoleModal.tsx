import { useState } from 'react'
import { X } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { useRoles, useAssignRole, useRemoveRole } from '@/hooks/useUsers'
import type { User } from '@/types'

interface AssignRoleModalProps {
  open: boolean
  onClose: () => void
  user: User
}

export function AssignRoleModal({ open, onClose, user }: AssignRoleModalProps) {
  const { data: allRoles = [] } = useRoles()
  const assignRole = useAssignRole()
  const removeRole = useRemoveRole()
  const [selectedRoleId, setSelectedRoleId] = useState('')

  const userRoleIds = new Set(user.roles.map((r) => r.id))
  const availableRoles = allRoles.filter((r) => !userRoleIds.has(r.id))

  const handleAssign = async () => {
    if (!selectedRoleId) return
    await assignRole.mutateAsync({ userId: user.id, roleId: selectedRoleId })
    setSelectedRoleId('')
  }

  const handleRemove = async (roleId: string) => {
    await removeRole.mutateAsync({ userId: user.id, roleId })
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Manage Roles"
      size="md"
      footer={
        <Button variant="secondary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <div className="space-y-4">
        <div>
          <p className="text-sm text-slate-500 mb-3">
            Managing roles for <span className="font-medium text-slate-800">{user.email}</span>
          </p>

          <div>
            <p className="text-xs font-semibold text-slate-500 uppercase mb-2">Current Roles</p>
            {user.roles.length === 0 ? (
              <p className="text-sm text-slate-400 italic">No roles assigned</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {user.roles.map((role) => (
                  <span
                    key={role.id}
                    className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-indigo-50 text-indigo-700 text-sm border border-indigo-200"
                  >
                    {role.name}
                    <button
                      onClick={() => handleRemove(role.id)}
                      disabled={removeRole.isPending}
                      className="text-indigo-400 hover:text-indigo-700 transition-colors"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="border-t border-slate-100 pt-4">
          <p className="text-xs font-semibold text-slate-500 uppercase mb-2">Add Role</p>
          {availableRoles.length === 0 ? (
            <p className="text-sm text-slate-400 italic">All available roles are already assigned</p>
          ) : (
            <div className="flex gap-2">
              <select
                value={selectedRoleId}
                onChange={(e) => setSelectedRoleId(e.target.value)}
                className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">Select a role...</option>
                {availableRoles.map((role) => (
                  <option key={role.id} value={role.id}>
                    {role.name}
                  </option>
                ))}
              </select>
              <Button
                onClick={handleAssign}
                disabled={!selectedRoleId}
                loading={assignRole.isPending}
                size="sm"
              >
                Add
              </Button>
            </div>
          )}
        </div>
      </div>
    </Modal>
  )
}
