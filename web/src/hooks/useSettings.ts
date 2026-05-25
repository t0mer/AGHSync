import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch, type AnyCredentials, type Settings } from '@/lib/api'

export function useSettings(credentials: AnyCredentials | null) {
  const qc = useQueryClient()

  const { data, isLoading, error } = useQuery<Settings>({
    queryKey: ['settings'],
    queryFn: () => apiFetch<Settings>('/api/v1/settings', { credentials }),
  })

  const updateUIAuth = useMutation({
    mutationFn: (body: { enabled: boolean; username?: string; password?: string }) =>
      apiFetch<Settings>('/api/v1/settings/ui-auth', {
        credentials,
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })

  const generateToken = useMutation({
    mutationFn: () =>
      apiFetch<{ token: string }>('/api/v1/settings/api-token', {
        credentials,
        method: 'POST',
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })

  const deleteToken = useMutation({
    mutationFn: () =>
      apiFetch('/api/v1/settings/api-token', { credentials, method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['settings'] }),
  })

  return {
    settings: data ?? null,
    isLoading,
    error,
    updateUIAuth,
    generateToken,
    deleteToken,
  }
}
