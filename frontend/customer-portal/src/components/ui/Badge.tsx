import { clsx } from 'clsx'
import type { CaseStatus, CasePriority } from '@/types'

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface BadgeProps {
  variant?: BadgeVariant
  children: React.ReactNode
  className?: string
}

const variantClasses: Record<BadgeVariant, string> = {
  default: 'bg-gray-100 text-gray-700',
  success: 'bg-green-100 text-green-700',
  warning: 'bg-yellow-100 text-yellow-700',
  danger: 'bg-red-100 text-red-700',
  info: 'bg-blue-100 text-blue-700',
}

export function Badge({ variant = 'default', children, className }: BadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </span>
  )
}

export function CaseStatusBadge({ status }: { status: CaseStatus }) {
  const variantMap: Record<CaseStatus, BadgeVariant> = {
    new: 'info',
    open: 'info',
    in_progress: 'warning',
    pending: 'warning',
    resolved: 'success',
    closed: 'default',
  }

  const labelMap: Record<CaseStatus, string> = {
    new: 'New',
    open: 'Open',
    in_progress: 'In Progress',
    pending: 'Pending',
    resolved: 'Resolved',
    closed: 'Closed',
  }

  return <Badge variant={variantMap[status]}>{labelMap[status]}</Badge>
}

export function CasePriorityBadge({ priority }: { priority: CasePriority }) {
  const variantMap: Record<CasePriority, BadgeVariant> = {
    low: 'default',
    medium: 'info',
    high: 'warning',
    critical: 'danger',
  }

  const labelMap: Record<CasePriority, string> = {
    low: 'Low',
    medium: 'Medium',
    high: 'High',
    critical: 'Critical',
  }

  return <Badge variant={variantMap[priority]}>{labelMap[priority]}</Badge>
}
