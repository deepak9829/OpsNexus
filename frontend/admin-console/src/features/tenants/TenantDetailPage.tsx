import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Building2 } from 'lucide-react'
import { useTenant, useTenantSettings, useTenantMembers } from '@/hooks/useTenants'
import { Badge, planBadge, statusBadge } from '@/components/ui/Badge'
import { Card, CardHeader, CardContent } from '@/components/ui/Card'
import { Table } from '@/components/ui/Table'
import { Spinner } from '@/components/ui/Spinner'
import { TenantSettingsForm } from './TenantSettingsForm'
import { formatDate } from '@/utils/format'
import type { User } from '@/types'
import { cn } from '@/utils/cn'

type Tab = 'overview' | 'members' | 'settings' | 'organizations'

export function TenantDetailPage() {
  const { id = '' } = useParams()
  const [activeTab, setActiveTab] = useState<Tab>('overview')

  const { data: tenant, isLoading } = useTenant(id)
  const { data: settings } = useTenantSettings(id)
  const { data: members } = useTenantMembers(id)

  if (isLoading) {
    return (
      <div className="flex justify-center items-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!tenant) {
    return (
      <div className="text-center py-24">
        <p className="text-slate-500">Tenant not found.</p>
        <Link to="/tenants" className="text-indigo-600 hover:underline text-sm mt-2 inline-block">
          Back to Tenants
        </Link>
      </div>
    )
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'Overview' },
    { key: 'members', label: `Members (${members?.meta.total ?? 0})` },
    { key: 'settings', label: 'Settings' },
    { key: 'organizations', label: 'Organizations' },
  ]

  const memberColumns = [
    {
      key: 'email',
      header: 'Email',
      render: (u: User) => <span className="font-medium text-slate-800">{u.email}</span>,
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
          {u.roles.map((r) => (
            <Badge key={r.id} variant="indigo">{r.name}</Badge>
          ))}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (u: User) => <Badge variant={statusBadge(u.status)}>{u.status}</Badge>,
    },
  ]

  return (
    <div className="space-y-6">
      {/* Back link + header */}
      <div>
        <Link to="/tenants" className="inline-flex items-center text-sm text-slate-500 hover:text-slate-700 mb-3">
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back to Tenants
        </Link>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-indigo-100 rounded-xl flex items-center justify-center">
              <Building2 className="h-6 w-6 text-indigo-600" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-slate-900">{tenant.name}</h1>
              <p className="text-slate-500 text-sm">{tenant.slug}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={planBadge(tenant.plan)}>{tenant.plan}</Badge>
            <Badge variant={statusBadge(tenant.status)}>{tenant.status}</Badge>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="flex gap-1 -mb-px">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'px-4 py-2.5 text-sm font-medium border-b-2 transition-colors',
                activeTab === tab.key
                  ? 'border-indigo-600 text-indigo-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700',
              )}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab content */}
      {activeTab === 'overview' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Card>
            <CardHeader title="Tenant Details" />
            <CardContent>
              <dl className="space-y-3">
                {[
                  { label: 'ID', value: tenant.id },
                  { label: 'Created', value: formatDate(tenant.createdAt) },
                  { label: 'Updated', value: formatDate(tenant.updatedAt) },
                  { label: 'Plan', value: tenant.plan },
                  { label: 'Max Users', value: settings?.maxUsers ?? '—' },
                ].map(({ label, value }) => (
                  <div key={label} className="flex justify-between text-sm">
                    <dt className="text-slate-500">{label}</dt>
                    <dd className="text-slate-900 font-medium">{String(value)}</dd>
                  </div>
                ))}
              </dl>
            </CardContent>
          </Card>
          {settings && (
            <Card>
              <CardHeader title="Active Features" />
              <CardContent>
                <div className="space-y-2">
                  {Object.entries(settings.features).map(([feat, enabled]) => (
                    <div key={feat} className="flex items-center justify-between text-sm">
                      <span className="text-slate-700 capitalize">{feat.replace(/_/g, ' ')}</span>
                      <Badge variant={enabled ? 'green' : 'gray'}>{enabled ? 'Enabled' : 'Disabled'}</Badge>
                    </div>
                  ))}
                  {Object.keys(settings.features).length === 0 && (
                    <p className="text-sm text-slate-400">No features configured</p>
                  )}
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {activeTab === 'members' && (
        <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
          <Table
            columns={memberColumns}
            data={members?.data ?? []}
            emptyMessage="No members found for this tenant."
          />
        </div>
      )}

      {activeTab === 'settings' && settings && (
        <Card>
          <CardHeader title="Tenant Settings" description="Configure limits, allowed domains, and feature flags." />
          <CardContent>
            <TenantSettingsForm tenantId={id} settings={settings} />
          </CardContent>
        </Card>
      )}

      {activeTab === 'organizations' && (
        <Card>
          <CardHeader title="Organizations" />
          <CardContent>
            <p className="text-sm text-slate-500">Organization hierarchy view coming soon.</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
