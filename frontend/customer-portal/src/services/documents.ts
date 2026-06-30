import { apiClient } from './api'
import type { Document, FormTemplate, PaginatedResponse } from '@/types'

export const documentsService = {
  list: async (params?: { page?: number; limit?: number }): Promise<PaginatedResponse<Document>> => {
    const { data } = await apiClient.get<PaginatedResponse<Document>>('/documents', { params })
    return data
  },
  get: async (id: string): Promise<Document> => {
    const { data } = await apiClient.get<{ data: Document }>(`/documents/${id}`)
    return data.data
  },
  upload: async (file: File, caseId?: string): Promise<Document> => {
    const formData = new FormData()
    formData.append('file', file)
    if (caseId) formData.append('caseId', caseId)
    const { data } = await apiClient.post<{ data: Document }>('/documents/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    return data.data
  },
  listForms: async (): Promise<FormTemplate[]> => {
    const { data } = await apiClient.get<{ data: FormTemplate[] }>('/forms')
    return data.data
  },
}
