import { describe, it, expect, vi } from 'vitest'
import { apiFetch, ApiError } from '@/lib/api'

describe('apiFetch', () => {
  it('sends request to the given path', async () => {
    const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    )
    await apiFetch('/api/v1/settings', { credentials: null })
    expect(spy).toHaveBeenCalledWith('/api/v1/settings', expect.objectContaining({ method: 'GET' }))
    spy.mockRestore()
  })

  it('attaches Authorization header when credentials provided', async () => {
    const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    )
    await apiFetch('/api/v1/settings', { credentials: { username: 'admin', password: 'secret' } })
    const headers = (spy.mock.calls[0][1] as RequestInit).headers as Record<string, string>
    expect(headers['Authorization']).toBe('Basic ' + btoa('admin:secret'))
    spy.mockRestore()
  })

  it('omits Authorization header when credentials is null', async () => {
    const spy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    )
    await apiFetch('/api/v1/settings', { credentials: null })
    const headers = (spy.mock.calls[0][1] as RequestInit).headers as Record<string, string>
    expect(headers['Authorization']).toBeUndefined()
    spy.mockRestore()
  })

  it('throws ApiError on non-2xx response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
    )
    await expect(apiFetch('/api/v1/instances/bad', { credentials: null })).rejects.toBeInstanceOf(ApiError)
  })

  it('ApiError carries status code', async () => {
    expect.assertions(2)
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'conflict' }), { status: 409 })
    )
    try {
      await apiFetch('/api/v1/sync/run', { credentials: null, method: 'POST' })
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError)
      expect((e as ApiError).status).toBe(409)
    }
  })

  it('dispatches auth-clear event on 401', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 })
    )
    const handler = vi.fn()
    window.addEventListener('auth-clear', handler)
    await expect(apiFetch('/api/v1/settings', { credentials: { username: 'bad', password: 'bad' } })).rejects.toBeInstanceOf(ApiError)
    expect(handler).toHaveBeenCalled()
    window.removeEventListener('auth-clear', handler)
  })
})
