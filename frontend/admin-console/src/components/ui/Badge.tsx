import { cn } from '@/utils/cn'

type BadgeVariant = 'gray' | 'green' | 'yellow' | 'red' | 'blue' | 'purple' | 'indigo' | 'orange'

const variantClasses: Record<BadgeVariant, string> = {
  gray: 'bg-slate-100 text-slate-700',
  green: 'bg-green-100 text-green-700',
  yellow: 'bg-yellow-100 text-yellow-700',
  red: 'bg-red-100 text-red-700',
  blue: 'bg-blue-100 text-blue-700',
  purple: 'bg-purple-100 text-purple-700',
  indigo: 'bg-indigo-100 text-indigo-700',
  orange: 'bg-orange-100 text-orange-700',
}

interface BadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
  className?: string
}

export function Badge({ children, variant = 'gray', className }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        variantClasses[variant],
        className,
      )}
    >
      {children}
    </span>
  )
}

export function planBadge(plan: string) {
  const map: Record<string, BadgeVariant> = { free: 'gray', pro: 'blue', enterprise: 'purple' }
  return map[plan] ?? 'gray'
}

export function statusBadge(status: string): BadgeVariant {
  const map: Record<string, BadgeVariant> = {
    active: 'green',
    inactive: 'gray',
    suspended: 'red',
    new: 'blue',
    open: 'indigo',
    in_progress: 'yellow',
    pending: 'orange',
    resolved: 'green',
    closed: 'gray',
  }
  return map[status] ?? 'gray'
}

export function priorityBadge(priority: string): BadgeVariant {
  const map: Record<string, BadgeVariant> = {
    low: 'gray',
    medium: 'yellow',
    high: 'orange',
    critical: 'red',
  }
  return map[priority] ?? 'gray'
}
