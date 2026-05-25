import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import React from 'react'
import { http, HttpResponse } from 'msw'
import { server } from './server'
import { useSyncStatus } from '@/hooks/useSyncStatus'

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

describe('useSyncStatus', () => {
  it('returns idle status when current is null', async () => {
    const { result } = renderHook(() => useSyncStatus(null), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.status).toBe('idle')
    expect(result.current.current).toBeNull()
  })

  it('returns running status when current run exists', async () => {
    server.use(
      http.get('/api/v1/sync/status', () =>
        HttpResponse.json({
          current: {
            run_id: 'abc',
            triggered_by: 'manual',
            started_at: '2026-01-01T00:00:00Z',
            status: 'running',
          },
          last: null,
        })
      )
    )
    const { result } = renderHook(() => useSyncStatus(null), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.status).toBe('running')
    expect(result.current.current?.run_id).toBe('abc')
  })
})
