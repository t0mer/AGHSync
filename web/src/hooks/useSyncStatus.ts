import { useQuery } from '@tanstack/react-query'
import { apiFetch, type AnyCredentials, type SyncStatusResponse } from '@/lib/api'

export function useSyncStatus(credentials: AnyCredentials | null) {
  const { data, isLoading, error } = useQuery<SyncStatusResponse>({
    queryKey: ['sync-status'],
    queryFn: () => apiFetch<SyncStatusResponse>('/api/v1/sync/status', { credentials }),
    refetchInterval: (query) => {
      const d = query.state.data
      return d?.current ? 3_000 : 30_000
    },
  })

  return {
    isLoading,
    error,
    current: data?.current ?? null,
    last: data?.last ?? null,
    status: data?.current ? 'running' as const : 'idle' as const,
  }
}
