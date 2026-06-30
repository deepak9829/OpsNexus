import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Globe, Shield, Puzzle, Check } from 'lucide-react'
import { Card, CardHeader, CardContent } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { cn } from '@/utils/cn'

type Tab = 'general' | 'security' | 'integrations'

interface GeneralFormData {
  platformName: string
  supportEmail: string
  maxTenantsPerPlan: number
}

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('general')
  const [saved, setSaved] = useState(false)

  const { register, handleSubmit, formState: { isSubmitting } } = useForm<GeneralFormData>({
    defaultValues: {
      platformName: 'OpsNexus',
      supportEmail: 'support@opsnexus.io',
      maxTenantsPerPlan: 100,
    },
  })

  const onSubmit = async (_data: GeneralFormData) => {
    // Simulate save
    await new Promise((r) => setTimeout(r, 800))
    setSaved(true)
    setTimeout(() => setSaved(false), 2500)
  }

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    { key: 'general', label: 'General', icon: <Globe className="h-4 w-4" /> },
    { key: 'security', label: 'Security', icon: <Shield className="h-4 w-4" /> },
    { key: 'integrations', label: 'Integrations', icon: <Puzzle className="h-4 w-4" /> },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Platform Settings</h1>
        <p className="text-slate-500 text-sm mt-0.5">Manage global platform configuration</p>
      </div>

      {/* Tab navigation */}
      <div className="border-b border-slate-200">
        <nav className="flex gap-1 -mb-px">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={cn(
                'flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 transition-colors',
                activeTab === tab.key
                  ? 'border-indigo-600 text-indigo-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700',
              )}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* General tab */}
      {activeTab === 'general' && (
        <Card>
          <CardHeader title="General Settings" description="Platform-wide name and contact configuration." />
          <CardContent>
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4 max-w-md">
              <Input
                label="Platform Name"
                placeholder="OpsNexus"
                {...register('platformName')}
              />
              <Input
                label="Support Email"
                type="email"
                placeholder="support@opsnexus.io"
                {...register('supportEmail')}
              />
              <Input
                label="Max Tenants Per Plan (Enterprise)"
                type="number"
                min={1}
                {...register('maxTenantsPerPlan', { valueAsNumber: true })}
              />
              <div className="flex items-center gap-3">
                <Button type="submit" loading={isSubmitting}>Save Changes</Button>
                {saved && (
                  <span className="flex items-center gap-1 text-sm text-green-600 font-medium">
                    <Check className="h-4 w-4" />
                    Saved!
                  </span>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      {/* Security tab */}
      {activeTab === 'security' && (
        <div className="space-y-4">
          <Card>
            <CardHeader title="Session Policy" description="Controls for user session management." />
            <CardContent>
              <dl className="space-y-4 max-w-md">
                {[
                  { label: 'Session Timeout', value: '30 minutes of inactivity' },
                  { label: 'Max Concurrent Sessions', value: '3 per user' },
                  { label: 'Token Expiry (Access)', value: '15 minutes' },
                  { label: 'Token Expiry (Refresh)', value: '7 days' },
                ].map(({ label, value }) => (
                  <div key={label} className="flex justify-between py-2 border-b border-slate-100 last:border-0">
                    <dt className="text-sm text-slate-500">{label}</dt>
                    <dd className="text-sm font-medium text-slate-800">{value}</dd>
                  </div>
                ))}
              </dl>
              <p className="text-xs text-slate-400 mt-4">
                Session policy is configured via environment variables. Contact your infrastructure team to update.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader title="Password Policy" />
            <CardContent>
              <ul className="space-y-2">
                {[
                  'Minimum 12 characters',
                  'At least one uppercase letter',
                  'At least one lowercase letter',
                  'At least one number',
                  'At least one special character',
                  'Cannot reuse last 5 passwords',
                ].map((rule) => (
                  <li key={rule} className="flex items-center gap-2 text-sm text-slate-700">
                    <Check className="h-4 w-4 text-green-500 flex-shrink-0" />
                    {rule}
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Integrations tab */}
      {activeTab === 'integrations' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {[
            { name: 'Webhooks', description: 'Send real-time event notifications to external systems.', status: 'Coming Soon' },
            { name: 'Slack', description: 'Post case updates and alerts to Slack channels.', status: 'Coming Soon' },
            { name: 'PagerDuty', description: 'Trigger incidents for critical SLA breaches.', status: 'Coming Soon' },
            { name: 'SAML / SSO', description: 'Enable single sign-on with your identity provider.', status: 'Coming Soon' },
            { name: 'Email (SMTP)', description: 'Configure outgoing email server settings.', status: 'Coming Soon' },
            { name: 'SMS (Twilio)', description: 'Send SMS notifications for critical alerts.', status: 'Coming Soon' },
          ].map((integration) => (
            <Card key={integration.name} className="p-4">
              <div className="flex items-start justify-between mb-2">
                <h3 className="font-semibold text-slate-800">{integration.name}</h3>
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500">
                  {integration.status}
                </span>
              </div>
              <p className="text-sm text-slate-500">{integration.description}</p>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
