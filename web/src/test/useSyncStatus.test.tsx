import { renderHook, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
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

  it('invalidates history cache when run transitions from idle to running', async () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const spy = vi.spyOn(qc, 'invalidateQueries')
    const wrap = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useSyncStatus(null), { wrapper: wrap })
    // Wait for the initial fetch to complete (not just the first render) so the
    // effect has settled prevCurrent before we trigger the transition.
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.status).toBe('idle')

    // Transition to running
    server.use(
      http.get('/api/v1/sync/status', () =>
        HttpResponse.json({
          current: {
            run_id: 'xyz',
            triggered_by: 'watchdog',
            started_at: '2026-01-01T00:00:00Z',
            status: 'running',
          },
          last: null,
        })
      )
    )

    // Force the hook to re-fetch sync status and detect the transition
    await act(async () => {
      await qc.invalidateQueries({ queryKey: ['sync-status'] })
    })

    // Wait for both: status running AND history invalidation fired
    await waitFor(() => {
      expect(result.current.status).toBe('running')
      const historyCalls = spy.mock.calls.filter(
        ([arg]) => JSON.stringify(arg) === JSON.stringify({ queryKey: ['history'] })
      )
      expect(historyCalls).toHaveLength(1)
    })
  })

  it('invalidates history cache when run transitions from running to idle', async () => {
    // Start with a running sync
    server.use(
      http.get('/api/v1/sync/status', () =>
        HttpResponse.json({
          current: {
            run_id: 'abc',
            triggered_by: 'watchdog',
            started_at: '2026-01-01T00:00:00Z',
            status: 'running',
          },
          last: null,
        })
      )
    )

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const spy = vi.spyOn(qc, 'invalidateQueries')
    const wrap = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    )

    const { result } = renderHook(() => useSyncStatus(null), { wrapper: wrap })
    await waitFor(() => expect(result.current.status).toBe('running'))
    spy.mockClear() // Reset so we only count calls from the running→idle transition

    // Transition to idle
    server.use(
      http.get('/api/v1/sync/status', () =>
        HttpResponse.json({ current: null, last: null })
      )
    )

    // Force the hook to re-fetch sync status and detect the transition
    await act(async () => {
      await qc.invalidateQueries({ queryKey: ['sync-status'] })
    })

    await waitFor(() => {
      expect(result.current.status).toBe('idle')
      const historyCalls = spy.mock.calls.filter(
        ([arg]) => JSON.stringify(arg) === JSON.stringify({ queryKey: ['history'] })
      )
      expect(historyCalls).toHaveLength(1)
    })
  })
})
