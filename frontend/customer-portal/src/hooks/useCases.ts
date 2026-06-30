import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { casesService, type CreateCasePayload } from '@/services/cases'

export function useCases(params?: { page?: number; status?: string; priority?: string }) {
  return useQuery({
    queryKey: ['cases', params],
    queryFn: () => casesService.list(params),
  })
}

export function useCase(id: string) {
  return useQuery({
    queryKey: ['cases', id],
    queryFn: () => casesService.get(id),
    enabled: !!id,
  })
}

export function useCaseTasks(caseId: string) {
  return useQuery({
    queryKey: ['cases', caseId, 'tasks'],
    queryFn: () => casesService.listTasks(caseId),
    enabled: !!caseId,
  })
}

export function useCaseComments(caseId: string) {
  return useQuery({
    queryKey: ['cases', caseId, 'comments'],
    queryFn: () => casesService.listComments(caseId),
    enabled: !!caseId,
  })
}

export function useCreateCase() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateCasePayload) => casesService.create(payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}

export function useUpdateCase() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<CreateCasePayload> }) =>
      casesService.update(id, payload),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ['cases', id] })
      qc.invalidateQueries({ queryKey: ['cases'] })
    },
  })
}

export function useTransitionCase() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, toStatus, reason }: { id: string; toStatus: string; reason?: string }) =>
      casesService.transition(id, toStatus, reason),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ['cases', id] })
      qc.invalidateQueries({ queryKey: ['cases'] })
    },
  })
}

export function useAddComment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ caseId, body }: { caseId: string; body: string }) =>
      casesService.addComment(caseId, body),
    onSuccess: (_, { caseId }) => {
      qc.invalidateQueries({ queryKey: ['cases', caseId, 'comments'] })
    },
  })
}
