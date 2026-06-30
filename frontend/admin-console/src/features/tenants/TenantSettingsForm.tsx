import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { X, Plus } from 'lucide-react'
import type { TenantSettings } from '@/types'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { useUpdateTenantSettings } from '@/hooks/useTenants'

interface TenantSettingsFormProps {
  tenantId: string
  settings: TenantSettings
}

const DEFAULT_FEATURES = ['sso', 'api_access', 'audit_log', 'advanced_workflows', 'bulk_operations']

export function TenantSettingsForm({ tenantId, settings }: TenantSettingsFormProps) {
  const updateSettings = useUpdateTenantSettings()
  const [domains, setDomains] = useState<string[]>(settings.allowedDomains ?? [])
  const [newDomain, setNewDomain] = useState('')
  const [features, setFeatures] = useState<Record<string, boolean>>(settings.features ?? {})
  const [saved, setSaved] = useState(false)

  const { register, handleSubmit, formState: { isSubmitting } } = useForm({
    defaultValues: {
      maxUsers: settings.maxUsers,
      emailEnabled: settings.notificationPrefs?.emailEnabled ?? true,
      smsEnabled: settings.notificationPrefs?.smsEnabled ?? false,
      inAppEnabled: settings.notificationPrefs?.inAppEnabled ?? true,
    },
  })

  const addDomain = () => {
    if (newDomain && !domains.includes(newDomain)) {
      setDomains((d) => [...d, newDomain])
      setNewDomain('')
    }
  }

  const removeDomain = (d: string) => setDomains((prev) => prev.filter((x) => x !== d))

  const toggleFeature = (feature: string) => {
    setFeatures((f) => ({ ...f, [feature]: !f[feature] }))
  }

  const onSubmit = async (data: { maxUsers: number; emailEnabled: boolean; smsEnabled: boolean; inAppEnabled: boolean }) => {
    await updateSettings.mutateAsync({
      id: tenantId,
      settings: {
        maxUsers: Number(data.maxUsers),
        allowedDomains: domains,
        features,
        notificationPrefs: {
          emailEnabled: data.emailEnabled,
          smsEnabled: data.smsEnabled,
          inAppEnabled: data.inAppEnabled,
        },
      },
    })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Max users */}
      <div>
        <Input
          label="Max Users"
          type="number"
          min={1}
          max={10000}
          {...register('maxUsers', { valueAsNumber: true })}
        />
      </div>

      {/* Allowed Domains */}
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-2">Allowed Email Domains</label>
        <div className="flex gap-2 mb-2">
          <input
            type="text"
            value={newDomain}
            onChange={(e) => setNewDomain(e.target.value)}
            placeholder="company.com"
            className="flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
            onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addDomain() } }}
          />
          <Button type="button" variant="secondary" size="sm" onClick={addDomain}>
            <Plus className="h-4 w-4" />
          </Button>
        </div>
        <div className="flex flex-wrap gap-2">
          {domains.map((d) => (
            <span key={d} className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full bg-slate-100 text-slate-700 text-xs">
              {d}
              <button type="button" onClick={() => removeDomain(d)} className="text-slate-400 hover:text-slate-600">
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
          {domains.length === 0 && (
            <p className="text-xs text-slate-400">No domains configured — all domains allowed</p>
          )}
        </div>
      </div>

      {/* Features */}
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-2">Feature Toggles</label>
        <div className="space-y-2">
          {DEFAULT_FEATURES.map((feature) => (
            <label key={feature} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={features[feature] ?? false}
                onChange={() => toggleFeature(feature)}
                className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-sm text-slate-700 capitalize">{feature.replace(/_/g, ' ')}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Notification Prefs */}
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-2">Notification Preferences</label>
        <div className="space-y-2">
          {[
            { field: 'emailEnabled' as const, label: 'Email Notifications' },
            { field: 'smsEnabled' as const, label: 'SMS Notifications' },
            { field: 'inAppEnabled' as const, label: 'In-App Notifications' },
          ].map(({ field, label }) => (
            <label key={field} className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                {...register(field)}
                className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              <span className="text-sm text-slate-700">{label}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="flex items-center gap-3">
        <Button type="submit" loading={isSubmitting}>Save Settings</Button>
        {saved && <span className="text-sm text-green-600 font-medium">Saved!</span>}
      </div>
    </form>
  )
}
