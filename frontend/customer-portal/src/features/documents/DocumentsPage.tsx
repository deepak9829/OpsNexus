import { useRef, useState } from 'react'
import { Upload, FileText, ChevronLeft, ChevronRight } from 'lucide-react'
import { useDocuments, useUploadDocument } from '@/hooks/useDocuments'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { PageSpinner } from '@/components/ui/Spinner'
import { EmptyState } from '@/components/ui/EmptyState'
import { formatDate, formatBytes } from '@/utils/format'

const MIME_ICONS: Record<string, string> = {
  'application/pdf': '📄',
  'image/png': '🖼️',
  'image/jpeg': '🖼️',
  'text/plain': '📝',
  'application/zip': '🗜️',
}

export function DocumentsPage() {
  const [page, setPage] = useState(1)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)

  const { data, isLoading, isError } = useDocuments(page)
  const uploadDocument = useUploadDocument()

  const documents = data?.data ?? []
  const meta = data?.meta

  const handleUploadClick = () => {
    setUploadError(null)
    fileInputRef.current?.click()
  }

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const maxSize = 50 * 1024 * 1024 // 50 MB
    if (file.size > maxSize) {
      setUploadError('File size exceeds 50 MB limit.')
      return
    }

    try {
      await uploadDocument.mutateAsync({ file })
    } catch {
      setUploadError('Upload failed. Please try again.')
    } finally {
      // Reset input so same file can be re-uploaded
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Documents</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            {meta ? `${meta.total} document${meta.total !== 1 ? 's' : ''}` : ''}
          </p>
        </div>
        <Button onClick={handleUploadClick} loading={uploadDocument.isPending}>
          <Upload className="h-4 w-4" />
          Upload File
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleFileChange}
          accept="*/*"
        />
      </div>

      {/* Upload progress / error */}
      {uploadDocument.isPending && (
        <div className="rounded-md bg-blue-50 border border-blue-200 px-4 py-3 text-sm text-blue-700">
          Uploading file, please wait...
        </div>
      )}
      {uploadDocument.isSuccess && (
        <div className="rounded-md bg-green-50 border border-green-200 px-4 py-3 text-sm text-green-700">
          File uploaded successfully!
        </div>
      )}
      {uploadError && (
        <div className="rounded-md bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
          {uploadError}
        </div>
      )}

      {/* Documents table */}
      <Card>
        {isLoading ? (
          <PageSpinner />
        ) : isError ? (
          <div className="py-8 text-center text-sm text-red-600">
            Failed to load documents.
          </div>
        ) : documents.length === 0 ? (
          <EmptyState
            icon={<FileText className="h-8 w-8" />}
            title="No documents yet"
            description="Upload your first document to get started."
            action={{ label: 'Upload File', onClick: handleUploadClick }}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    {['File', 'Type', 'Size', 'Versions', 'Uploaded'].map((h) => (
                      <th
                        key={h}
                        className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {documents.map((doc) => (
                    <tr key={doc.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <span className="text-lg" aria-hidden="true">
                            {MIME_ICONS[doc.mimeType] ?? '📎'}
                          </span>
                          <span className="text-sm font-medium text-gray-900 truncate max-w-[200px]">
                            {doc.filename}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500 font-mono">
                        {doc.mimeType}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500">
                        {formatBytes(doc.sizeBytes)}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500 text-center">
                        {doc.versionCount}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500">
                        {formatDate(doc.createdAt)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {meta && meta.totalPages > 1 && (
              <div className="flex items-center justify-between px-4 py-3 border-t border-gray-200">
                <p className="text-sm text-gray-500">
                  Page {meta.page} of {meta.totalPages}
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Previous
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setPage((p) => p + 1)}
                    disabled={page >= meta.totalPages}
                  >
                    Next
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  )
}
