import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'
import React from 'react'
import { useInstances } from '@/hooks/useInstances'

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

describe('useInstances', () => {
  it('returns instances list', async () => {
    const { result } = renderHook(() => useInstances(null), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(result.current.instances).toHaveLength(1)
    expect(result.current.instances[0].name).toBe('Master')
  })

  it('exposes createInstance mutation', () => {
    const { result } = renderHook(() => useInstances(null), { wrapper })
    expect(typeof result.current.createInstance.mutate).toBe('function')
  })

  it('exposes deleteInstance mutation', () => {
    const { result } = renderHook(() => useInstances(null), { wrapper })
    expect(typeof result.current.deleteInstance.mutate).toBe('function')
  })
})
