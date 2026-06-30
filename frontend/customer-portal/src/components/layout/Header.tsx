import { useState, useRef, useEffect } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Bell, ChevronRight, LogOut, User } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { useNotifications } from '@/hooks/useNotifications'

const routeLabels: Record<string, string> = {
  '': 'Dashboard',
  cases: 'My Cases',
  documents: 'Documents',
  forms: 'Forms',
  notifications: 'Notifications',
  profile: 'Profile',
}

function Breadcrumb() {
  const location = useLocation()
  const segments = location.pathname.split('/').filter(Boolean)

  if (segments.length === 0) {
    return <span className="text-sm font-medium text-gray-900">Dashboard</span>
  }

  return (
    <nav className="flex items-center gap-1" aria-label="Breadcrumb">
      <Link to="/" className="text-sm text-gray-500 hover:text-gray-700">
        Dashboard
      </Link>
      {segments.map((seg, idx) => {
        const isLast = idx === segments.length - 1
        const label = routeLabels[seg] ?? seg
        const path = '/' + segments.slice(0, idx + 1).join('/')

        return (
          <span key={path} className="flex items-center gap-1">
            <ChevronRight className="h-4 w-4 text-gray-400" />
            {isLast ? (
              <span className="text-sm font-medium text-gray-900 capitalize">{label}</span>
            ) : (
              <Link to={path} className="text-sm text-gray-500 hover:text-gray-700 capitalize">
                {label}
              </Link>
            )}
          </span>
        )
      })}
    </nav>
  )
}

export function Header() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { data: notificationsData } = useNotifications(1)
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const unreadCount = notificationsData?.data.filter((n) => !n.read).length ?? 0

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleLogout = async () => {
    setDropdownOpen(false)
    await logout()
    navigate('/login')
  }

  return (
    <header className="sticky top-0 z-20 flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6 shadow-sm">
      {/* Breadcrumb */}
      <Breadcrumb />

      {/* Right side actions */}
      <div className="flex items-center gap-4">
        {/* Notification bell */}
        <Link
          to="/notifications"
          className="relative p-2 rounded-md text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors"
          aria-label={`Notifications${unreadCount > 0 ? `, ${unreadCount} unread` : ''}`}
        >
          <Bell className="h-5 w-5" />
          {unreadCount > 0 && (
            <span className="absolute -top-0.5 -right-0.5 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs font-bold text-white">
              {unreadCount > 99 ? '99+' : unreadCount}
            </span>
          )}
        </Link>

        {/* User dropdown */}
        <div className="relative" ref={dropdownRef}>
          <button
            onClick={() => setDropdownOpen(!dropdownOpen)}
            className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-gray-700 hover:bg-gray-100 transition-colors"
          >
            <div className="flex h-7 w-7 items-center justify-center rounded-full bg-blue-600 text-xs font-semibold text-white">
              {user?.firstName.charAt(0).toUpperCase()}
              {user?.lastName.charAt(0).toUpperCase()}
            </div>
            <span className="font-medium">
              {user?.firstName} {user?.lastName}
            </span>
          </button>

          {dropdownOpen && (
            <div className="absolute right-0 mt-1 w-48 rounded-md border border-gray-200 bg-white py-1 shadow-lg z-50">
              <Link
                to="/profile"
                onClick={() => setDropdownOpen(false)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
              >
                <User className="h-4 w-4" />
                Profile
              </Link>
              <hr className="my-1 border-gray-200" />
              <button
                onClick={handleLogout}
                className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50"
              >
                <LogOut className="h-4 w-4" />
                Sign Out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  )
}
