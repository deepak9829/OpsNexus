import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  FolderOpen,
  FileText,
  ClipboardList,
  Bell,
  User,
  Zap,
} from 'lucide-react'
import { clsx } from 'clsx'
import { useAuth } from '@/hooks/useAuth'

interface NavItem {
  label: string
  to: string
  icon: React.ReactNode
}

const navItems: NavItem[] = [
  { label: 'Dashboard', to: '/', icon: <LayoutDashboard className="h-5 w-5" /> },
  { label: 'My Cases', to: '/cases', icon: <FolderOpen className="h-5 w-5" /> },
  { label: 'Documents', to: '/documents', icon: <FileText className="h-5 w-5" /> },
  { label: 'Forms', to: '/forms', icon: <ClipboardList className="h-5 w-5" /> },
  { label: 'Notifications', to: '/notifications', icon: <Bell className="h-5 w-5" /> },
  { label: 'Profile', to: '/profile', icon: <User className="h-5 w-5" /> },
]

export function Sidebar() {
  const { user } = useAuth()

  return (
    <aside className="flex h-screen w-60 flex-col bg-gray-900 text-white fixed left-0 top-0 z-30">
      {/* Brand */}
      <div className="flex items-center gap-2 px-6 py-5 border-b border-gray-700">
        <div className="flex h-8 w-8 items-center justify-center rounded-md bg-blue-600">
          <Zap className="h-5 w-5 text-white" />
        </div>
        <span className="text-lg font-bold tracking-tight">OpsNexus</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto py-4 px-3">
        <ul className="space-y-1">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-blue-600 text-white'
                      : 'text-gray-300 hover:bg-gray-800 hover:text-white',
                  )
                }
              >
                {item.icon}
                {item.label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      {/* User info */}
      {user && (
        <div className="border-t border-gray-700 px-4 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-500 text-sm font-semibold text-white">
              {user.firstName.charAt(0).toUpperCase()}
              {user.lastName.charAt(0).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium text-white">
                {user.firstName} {user.lastName}
              </p>
              <p className="truncate text-xs text-gray-400">{user.email}</p>
            </div>
          </div>
        </div>
      )}
    </aside>
  )
}
