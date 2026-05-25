import { useQuery } from '@tanstack/react-query'
import { apiFetch, type AnyCredentials, type Run, type RunDetail } from '@/lib/api'

export function useHistory(credentials: AnyCredentials | null, limit = 20, offset = 0) {
  const { data, isLoading, error } = useQuery<Run[]>({
    queryKey: ['history', limit, offset],
    queryFn: () =>
      apiFetch<Run[]>(`/api/v1/history?limit=${limit}&offset=${offset}`, { credentials }),
  })
  return { runs: data ?? [], isLoading, error }
}

export function useHistoryRun(credentials: AnyCredentials | null, runId: string) {
  const { data, isLoading, error } = useQuery<RunDetail>({
    queryKey: ['history-run', runId],
    queryFn: () => apiFetch<RunDetail>(`/api/v1/history/${runId}`, { credentials }),
    enabled: !!runId,
  })
  return { run: data ?? null, isLoading, error }
}
