import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { documentsService } from '@/services/documents'

export function useDocuments(page = 1) {
  return useQuery({
    queryKey: ['documents', page],
    queryFn: () => documentsService.list({ page, limit: 20 }),
  })
}

export function useForms() {
  return useQuery({
    queryKey: ['forms'],
    queryFn: () => documentsService.listForms(),
  })
}

export function useUploadDocument() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ file, caseId }: { file: File; caseId?: string }) =>
      documentsService.upload(file, caseId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['documents'] }),
  })
}
