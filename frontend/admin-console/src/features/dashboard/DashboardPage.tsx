import { Link } from 'react-router-dom'
import { Building2, Users, Briefcase, AlertTriangle, Plus, ScrollText } from 'lucide-react'
import { useTenants } from '@/hooks/useTenants'
import { useCases } from '@/hooks/useCases'
import { useAuditEvents } from '@/hooks/useAudit'
import { StatCard, Card, CardHeader, CardContent } from '@/components/ui/Card'
import { Badge, statusBadge, planBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { formatRelative } from '@/utils/format'

export function DashboardPage() {
  const { data: tenantsData, isLoading: tenantsLoading } = useTenants({ limit: 100 })
  const { data: allCases, isLoading: casesLoading } = useCases({ limit: 100 })
  const { data: openCases } = useCases({ status: 'open', limit: 100 })
  const { data: auditData, isLoading: auditLoading } = useAuditEvents({ limit: 10 })

  const activeTenants = tenantsData?.data.filter((t) => t.status === 'active').length ?? 0
  const totalTenants = tenantsData?.meta.total ?? 0
  const totalCases = allCases?.meta.total ?? 0
  const openCasesCount = openCases?.meta.total ?? 0

  const isLoading = tenantsLoading || casesLoading || auditLoading

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Dashboard</h1>
          <p className="text-slate-500 text-sm mt-0.5">Platform overview across all tenants</p>
        </div>
        <div className="flex gap-2">
          <Link to="/tenants">
            <Button variant="secondary" size="sm">
              <Building2 className="h-4 w-4 mr-1.5" />
              View Tenants
            </Button>
          </Link>
          <Link to="/audit">
            <Button variant="indigo" size="sm">
              <ScrollText className="h-4 w-4 mr-1.5" />
              Audit Log
            </Button>
          </Link>
        </div>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Total Tenants"
          value={isLoading ? '—' : totalTenants}
          icon={<Building2 className="h-5 w-5" />}
        />
        <StatCard
          label="Active Tenants"
          value={isLoading ? '—' : activeTenants}
          icon={<Building2 className="h-5 w-5" />}
          trend={{ value: `${totalTenants ? Math.round((activeTenants / totalTenants) * 100) : 0}% active`, positive: true }}
        />
        <StatCard
          label="Total Cases"
          value={isLoading ? '—' : totalCases}
          icon={<Briefcase className="h-5 w-5" />}
        />
        <StatCard
          label="Open Cases"
          value={isLoading ? '—' : openCasesCount}
          icon={<AlertTriangle className="h-5 w-5" />}
          trend={{ value: openCasesCount > 0 ? 'Needs attention' : 'All clear', positive: openCasesCount === 0 }}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Tenant health table */}
        <Card>
          <CardHeader
            title="Tenant Health"
            action={
              <Link to="/tenants">
                <Button variant="ghost" size="sm">View all</Button>
              </Link>
            }
          />
          <CardContent className="p-0">
            {tenantsLoading ? (
              <div className="flex justify-center py-8"><Spinner /></div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-slate-100">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase">Tenant</th>
                      <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase">Plan</th>
                      <th className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-50">
                    {tenantsData?.data.slice(0, 6).map((tenant) => (
                      <tr key={tenant.id} className="hover:bg-slate-50">
                        <td className="px-4 py-3">
                          <Link to={`/tenants/${tenant.id}`} className="text-sm font-medium text-indigo-600 hover:text-indigo-800">
                            {tenant.name}
                          </Link>
                          <p className="text-xs text-slate-400">{tenant.slug}</p>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant={planBadge(tenant.plan)}>{tenant.plan}</Badge>
                        </td>
                        <td className="px-4 py-3">
                          <Badge variant={statusBadge(tenant.status)}>{tenant.status}</Badge>
                        </td>
                      </tr>
                    ))}
                    {(!tenantsData?.data.length) && (
                      <tr>
                        <td colSpan={3} className="px-4 py-8 text-center text-sm text-slate-400">No tenants found</td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Activity */}
        <Card>
          <CardHeader
            title="Recent Activity"
            action={
              <Link to="/audit">
                <Button variant="ghost" size="sm">View all</Button>
              </Link>
            }
          />
          <CardContent className="p-0">
            {auditLoading ? (
              <div className="flex justify-center py-8"><Spinner /></div>
            ) : (
              <ul className="divide-y divide-slate-100">
                {auditData?.data.map((event) => (
                  <li key={event.id} className="px-4 py-3 hover:bg-slate-50">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="text-sm text-slate-800 truncate">
                          <span className="font-medium">{event.actorEmail}</span>
                          {' '}
                          <span className="text-indigo-600">{event.action}</span>
                          {' '}
                          <span className="text-slate-500">{event.resource}</span>
                        </p>
                      </div>
                      <span className="text-xs text-slate-400 flex-shrink-0">{formatRelative(event.timestamp)}</span>
                    </div>
                  </li>
                ))}
                {!auditData?.data.length && (
                  <li className="px-4 py-8 text-center text-sm text-slate-400">No recent activity</li>
                )}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Quick actions */}
      <Card>
        <CardHeader title="Quick Actions" />
        <CardContent>
          <div className="flex flex-wrap gap-3">
            <Link to="/tenants">
              <Button variant="indigo">
                <Plus className="h-4 w-4 mr-1.5" />
                Create Tenant
              </Button>
            </Link>
            <Link to="/audit">
              <Button variant="secondary">
                <ScrollText className="h-4 w-4 mr-1.5" />
                View Audit Log
              </Button>
            </Link>
            <Link to="/users">
              <Button variant="secondary">
                <Users className="h-4 w-4 mr-1.5" />
                Manage Users
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
