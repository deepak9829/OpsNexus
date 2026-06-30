import { format, formatDistanceToNow } from 'date-fns'

export const formatDate = (date: string) => format(new Date(date), 'MMM d, yyyy')
export const formatDateTime = (date: string) => format(new Date(date), 'MMM d, yyyy HH:mm')
export const formatRelative = (date: string) => formatDistanceToNow(new Date(date), { addSuffix: true })
export const formatBytes = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 ** 2).toFixed(1)} MB`
}
