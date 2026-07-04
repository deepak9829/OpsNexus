import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ChevronRight, MessageSquare, Tag, Calendar, User, ArrowRight } from 'lucide-react'
import { useCase, useCaseTasks, useCaseComments, useTransitionCase, useAddComment, useUpdateCase } from '@/hooks/useCases'
import { CaseStatusBadge, CasePriorityBadge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardHeader, CardBody } from '@/components/ui/Card'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { PageSpinner } from '@/components/ui/Spinner'
import { formatDate, formatDateTime, formatRelative } from '@/utils/format'
import type { CaseStatus } from '@/types'

const STATUS_TRANSITIONS: Record<CaseStatus, CaseStatus[]> = {
  new: ['open'],
  open: ['in_progress', 'pending', 'closed'],
  in_progress: ['pending', 'resolved'],
  pending: ['open', 'in_progress', 'resolved'],
  resolved: ['closed', 'open'],
  closed: [],
}

const TRANSITION_LABELS: Record<CaseStatus, string> = {
  new: 'New',
  open: 'Open',
  in_progress: 'Start Progress',
  pending: 'Mark Pending',
  resolved: 'Mark Resolved',
  closed: 'Close',
}

function TaskItem({ title, status }: { title: string; status: string }) {
  const statusColor: Record<string, string> = {
    todo: 'bg-gray-200',
    in_progress: 'bg-blue-400',
    done: 'bg-green-400',
    blocked: 'bg-red-400',
  }

  return (
    <div className="flex items-center gap-3 py-2">
      <span className={`h-2.5 w-2.5 flex-shrink-0 rounded-full ${statusColor[status] ?? 'bg-gray-300'}`} />
      <span className="text-sm text-gray-700 flex-1">{title}</span>
      <span className="text-xs text-gray-400 capitalize">{status.replace('_', ' ')}</span>
    </div>
  )
}

export function CaseDetailPage() {
  const { id } = useParams<{ id: string }>()
  const caseId = id ?? ''

  const { data: caseData, isLoading } = useCase(caseId)
  const { data: tasks } = useCaseTasks(caseId)
  const { data: comments } = useCaseComments(caseId)

  const transitionCase = useTransitionCase()
  const addComment = useAddComment()
  const updateCase = useUpdateCase()

  const [transitionModal, setTransitionModal] = useState(false)
  const [selectedStatus, setSelectedStatus] = useState<CaseStatus | ''>('')
  const [transitionReason, setTransitionReason] = useState('')

  const [editModal, setEditModal] = useState(false)
  const [editTitle, setEditTitle] = useState('')
  const [editDescription, setEditDescription] = useState('')

  const [commentText, setCommentText] = useState('')
  const [isSubmittingComment, setIsSubmittingComment] = useState(false)

  if (isLoading) return <PageSpinner />
  if (!caseData) {
    return (
      <div className="py-12 text-center text-sm text-gray-500">Case not found.</div>
    )
  }

  const availableTransitions = STATUS_TRANSITIONS[caseData.status] ?? []

  const handleOpenTransition = () => {
    setSelectedStatus('')
    setTransitionReason('')
    setTransitionModal(true)
  }

  const handleTransition = async () => {
    if (!selectedStatus) return
    await transitionCase.mutateAsync({ id: caseId, toStatus: selectedStatus, reason: transitionReason || undefined })
    setTransitionModal(false)
  }

  const handleOpenEdit = () => {
    setEditTitle(caseData.title)
    setEditDescription(caseData.description)
    setEditModal(true)
  }

  const handleEdit = async () => {
    await updateCase.mutateAsync({ id: caseId, payload: { title: editTitle, description: editDescription } })
    setEditModal(false)
  }

  const handleAddComment = async () => {
    if (!commentText.trim()) return
    setIsSubmittingComment(true)
    try {
      await addComment.mutateAsync({ caseId, body: commentText.trim() })
      setCommentText('')
    } finally {
      setIsSubmittingComment(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-1 text-sm">
        <Link to="/cases" className="text-gray-500 hover:text-gray-700">
          Cases
        </Link>
        <ChevronRight className="h-4 w-4 text-gray-400" />
        <span className="font-medium text-gray-900 font-mono">{caseData.caseNumber}</span>
      </nav>

      {/* Case header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <span className="text-xs font-mono font-semibold text-blue-600 bg-blue-50 px-2 py-0.5 rounded">
              {caseData.caseNumber}
            </span>
            <CaseStatusBadge status={caseData.status} />
            <CasePriorityBadge priority={caseData.priority} />
            {caseData.sla.breached && (
              <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-700">
                SLA Breached
              </span>
            )}
          </div>
          <h1 className="text-xl font-bold text-gray-900">{caseData.title}</h1>
        </div>

        <div className="flex gap-2 flex-shrink-0">
          {availableTransitions.length > 0 && (
            <Button variant="secondary" size="sm" onClick={handleOpenTransition}>
              <ArrowRight className="h-4 w-4" />
              Transition Status
            </Button>
          )}
          <Button size="sm" onClick={handleOpenEdit}>
            Edit Case
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Left column: description + tasks */}
        <div className="lg:col-span-2 space-y-4">
          {/* Description */}
          <Card>
            <CardHeader>
              <h2 className="text-base font-semibold text-gray-900">Description</h2>
            </CardHeader>
            <CardBody>
              <p className="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">
                {caseData.description || 'No description provided.'}
              </p>
            </CardBody>
          </Card>

          {/* Tasks */}
          <Card>
            <CardHeader>
              <h2 className="text-base font-semibold text-gray-900">
                Tasks {tasks && tasks.length > 0 && `(${tasks.length})`}
              </h2>
            </CardHeader>
            <CardBody>
              {!tasks || tasks.length === 0 ? (
                <p className="text-sm text-gray-500">No tasks assigned yet.</p>
              ) : (
                <div className="divide-y divide-gray-100">
                  {tasks.map((t) => (
                    <TaskItem key={t.id} title={t.title} status={t.status} />
                  ))}
                </div>
              )}
            </CardBody>
          </Card>

          {/* Comments */}
          <Card>
            <CardHeader>
              <div className="flex items-center gap-2">
                <MessageSquare className="h-4 w-4 text-gray-500" />
                <h2 className="text-base font-semibold text-gray-900">
                  Comments {comments && comments.length > 0 && `(${comments.length})`}
                </h2>
              </div>
            </CardHeader>
            <CardBody className="space-y-4">
              {/* Existing comments */}
              {comments && comments.length > 0 ? (
                <div className="space-y-4">
                  {comments.map((c) => (
                    <div key={c.id} className="flex gap-3">
                      <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-gray-200 text-xs font-semibold text-gray-600">
                        U
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-baseline gap-2">
                          <span className="text-sm font-medium text-gray-900">User</span>
                          <span className="text-xs text-gray-400">{formatRelative(c.createdAt)}</span>
                        </div>
                        <p className="mt-0.5 text-sm text-gray-700 whitespace-pre-wrap">{c.body}</p>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No comments yet. Be the first to comment.</p>
              )}

              {/* Add comment */}
              <div className="border-t border-gray-200 pt-4">
                <textarea
                  rows={3}
                  placeholder="Add a comment..."
                  value={commentText}
                  onChange={(e) => setCommentText(e.target.value)}
                  className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <div className="mt-2 flex justify-end">
                  <Button
                    size="sm"
                    onClick={handleAddComment}
                    loading={isSubmittingComment}
                    disabled={!commentText.trim()}
                  >
                    Add Comment
                  </Button>
                </div>
              </div>
            </CardBody>
          </Card>
        </div>

        {/* Right column: metadata */}
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <h2 className="text-sm font-semibold text-gray-900 uppercase tracking-wider">
                Details
              </h2>
            </CardHeader>
            <CardBody className="space-y-3 text-sm">
              <MetaRow icon={<Calendar className="h-4 w-4" />} label="Created" value={formatDateTime(caseData.createdAt)} />
              <MetaRow icon={<Calendar className="h-4 w-4" />} label="Updated" value={formatDateTime(caseData.updatedAt)} />
              {caseData.resolvedAt && (
                <MetaRow icon={<Calendar className="h-4 w-4" />} label="Resolved" value={formatDateTime(caseData.resolvedAt)} />
              )}
              {caseData.sla.dueAt && (
                <MetaRow
                  icon={<Calendar className="h-4 w-4" />}
                  label="SLA Due"
                  value={
                    <span className={caseData.sla.breached ? 'text-red-600 font-medium' : ''}>
                      {formatDate(caseData.sla.dueAt)}
                    </span>
                  }
                />
              )}
              {caseData.assigneeId && (
                <MetaRow icon={<User className="h-4 w-4" />} label="Assignee" value={caseData.assigneeId} />
              )}
              {caseData.tags.length > 0 && (
                <div className="flex items-start gap-2">
                  <Tag className="h-4 w-4 text-gray-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-xs text-gray-500 mb-1">Tags</p>
                    <div className="flex flex-wrap gap-1">
                      {caseData.tags.map((tag) => (
                        <span
                          key={tag}
                          className="inline-flex items-center rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600"
                        >
                          {tag}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
              )}
            </CardBody>
          </Card>
        </div>
      </div>

      {/* Transition modal */}
      <Modal isOpen={transitionModal} onClose={() => setTransitionModal(false)} title="Transition Case Status">
        <div className="space-y-4">
          <div>
            <p className="text-sm text-gray-600 mb-3">
              Current status: <CaseStatusBadge status={caseData.status} />
            </p>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              New Status <span className="text-red-500">*</span>
            </label>
            <select
              value={selectedStatus}
              onChange={(e) => setSelectedStatus(e.target.value as CaseStatus)}
              className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="">Select a status...</option>
              {availableTransitions.map((s) => (
                <option key={s} value={s}>
                  {TRANSITION_LABELS[s] ?? s}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Reason (optional)
            </label>
            <textarea
              rows={3}
              value={transitionReason}
              onChange={(e) => setTransitionReason(e.target.value)}
              placeholder="Why is this status changing?"
              className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setTransitionModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleTransition}
              disabled={!selectedStatus}
              loading={transitionCase.isPending}
            >
              Confirm Transition
            </Button>
          </div>
        </div>
      </Modal>

      {/* Edit modal */}
      <Modal isOpen={editModal} onClose={() => setEditModal(false)} title="Edit Case">
        <div className="space-y-4">
          <Input
            label="Title"
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            required
          />
          <div className="flex flex-col gap-1">
            <label className="block text-sm font-medium text-gray-700">Description</label>
            <textarea
              rows={5}
              value={editDescription}
              onChange={(e) => setEditDescription(e.target.value)}
              className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setEditModal(false)}>
              Cancel
            </Button>
            <Button onClick={handleEdit} loading={updateCase.isPending}>
              Save Changes
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

function MetaRow({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="flex items-start gap-2">
      <span className="text-gray-400 mt-0.5 flex-shrink-0">{icon}</span>
      <div className="min-w-0">
        <p className="text-xs text-gray-500">{label}</p>
        <p className="text-sm text-gray-900">{value}</p>
      </div>
    </div>
  )
}
