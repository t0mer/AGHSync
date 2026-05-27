import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch, type AnyCredentials, type SyncStatusResponse } from '@/lib/api'

export function useSyncStatus(credentials: AnyCredentials | null) {
  const qc = useQueryClient()
  // Tracks the previous value of `current` to detect run-state transitions.
  // Initialized to undefined (not null) so the initial mount (undefined → null)
  // is skipped. NOTE: != uses loose equality on purpose — undefined == null is
  // true in JS, so undefined != null is false.
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
    // Invalidate history on any run-state change (idle→running or running→idle)
    // so the History page refreshes both when a run appears and when it completes.
    // wasRunning uses != (loose equality) intentionally: undefined == null is true
    // in JS, so the initial mount (undefined → null) is correctly skipped.
    const wasRunning = prevCurrent.current != null
    const isRunning = current != null
    if (wasRunning !== isRunning) {
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
