import { Modal } from '@/components/ui/Modal'
import { formatDateTime } from '@/utils/format'
import type { AuditEvent } from '@/types'

interface AuditEventDetailModalProps {
  open: boolean
  onClose: () => void
  event: AuditEvent | null
}

function JsonDiff({ label, value }: { label: string; value?: Record<string, unknown> }) {
  if (!value || Object.keys(value).length === 0) return null
  return (
    <div>
      <p className="text-xs font-semibold text-slate-500 uppercase mb-1">{label}</p>
      <pre className="text-xs bg-slate-50 border border-slate-200 rounded-lg p-3 overflow-x-auto text-slate-700 leading-relaxed">
        {JSON.stringify(value, null, 2)}
      </pre>
    </div>
  )
}

export function AuditEventDetailModal({ open, onClose, event }: AuditEventDetailModalProps) {
  if (!event) return null

  return (
    <Modal open={open} onClose={onClose} title="Audit Event Details" size="xl">
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-3">
          {[
            { label: 'Event ID', value: event.id },
            { label: 'Timestamp', value: formatDateTime(event.timestamp) },
            { label: 'Actor', value: event.actorEmail },
            { label: 'Actor ID', value: event.actorId },
            { label: 'Action', value: event.action },
            { label: 'Resource', value: event.resource },
            { label: 'Resource ID', value: event.resourceId },
            { label: 'Tenant ID', value: event.tenantId },
            { label: 'IP Address', value: event.ipAddress },
          ].map(({ label, value }) => (
            <div key={label} className="bg-slate-50 rounded-lg p-3">
              <p className="text-xs font-semibold text-slate-500 uppercase mb-0.5">{label}</p>
              <p className="text-sm text-slate-800 font-mono break-all">{value}</p>
            </div>
          ))}
        </div>

        <div className="bg-slate-50 rounded-lg p-3">
          <p className="text-xs font-semibold text-slate-500 uppercase mb-0.5">User Agent</p>
          <p className="text-xs text-slate-600 font-mono break-all">{event.userAgent}</p>
        </div>

        {(event.oldValue || event.newValue) && (
          <div className="space-y-3 border-t border-slate-200 pt-4">
            <p className="text-sm font-semibold text-slate-700">Data Changes</p>
            <JsonDiff label="Before" value={event.oldValue} />
            <JsonDiff label="After" value={event.newValue} />
          </div>
        )}
      </div>
    </Modal>
  )
}
