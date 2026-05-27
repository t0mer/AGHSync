import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch, type AnyCredentials, type SyncStatusResponse } from '@/lib/api'

export function useSyncStatus(credentials: AnyCredentials | null) {
  const qc = useQueryClient()
  // Tracks the previous value of `current` to detect running→idle transitions.
  // Initialized to undefined (not null) so the first render (undefined → null) is
  // skipped by the != null guard below. NOTE: != uses loose equality on purpose —
  // undefined == null is true in JS, so undefined != null is false.
  const prevCurrent = useRef<SyncStatusResponse['current'] | undefined>(undefined)

  const { data, isLoading, error } = useQuery<SyncStatusResponse>({
    queryKey: ['sync-status'],
    queryFn: () => apiFetch<SyncStatusResponse>('/api/v1/sync/status', { credentials }),
    refetchInterval: (query) => {
      const d = query.state.data
      return d?.current ? 3_000 : 30_000
    },
  })

  const current = data?.current ?? null

  useEffect(() => {
    // prevCurrent.current != null uses loose equality intentionally:
    // undefined != null is false, so the initial mount (undefined → null) is skipped.
    // Only a genuine running→idle transition (object → null) fires the invalidation.
    if (prevCurrent.current != null && current === null) {
      qc.invalidateQueries({ queryKey: ['history'] })
    }
    prevCurrent.current = current
  }, [current, qc])

  return {
    isLoading,
    error,
    current,
    last: data?.last ?? null,
    status: current ? 'running' as const : 'idle' as const,
  }
}
