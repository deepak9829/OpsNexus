import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { casesService, type ListCasesParams } from '@/services/cases'
import type { Case } from '@/types'

export function useCases(params?: ListCasesParams) {
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

export function useBulkUpdateCases() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, payload }: { ids: string[]; payload: Partial<Pick<Case, 'status' | 'priority' | 'assigneeId'>> }) =>
      casesService.bulkUpdate(ids, payload),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['cases'] }),
  })
}
