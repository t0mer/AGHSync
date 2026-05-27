import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  apiFetch,
  type AnyCredentials,
  type NotificationChannel,
  type NotificationChannelPayload,
  type TestChannelPayload,
} from '@/lib/api'

export function useNotifications(credentials: AnyCredentials | null) {
  const qc = useQueryClient()

  const { data, isLoading, error } = useQuery<NotificationChannel[]>({
    queryKey: ['notifications'],
    queryFn: () => apiFetch<NotificationChannel[]>('/api/v1/notifications', { credentials }),
  })

  const createChannel = useMutation({
    mutationFn: (body: NotificationChannelPayload) =>
      apiFetch<NotificationChannel>('/api/v1/notifications', {
        credentials,
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })

  const updateChannel = useMutation({
    mutationFn: ({ id, ...body }: NotificationChannelPayload & { id: string }) =>
      apiFetch<NotificationChannel>(`/api/v1/notifications/${id}`, {
        credentials,
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })

  const deleteChannel = useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/v1/notifications/${id}`, { credentials, method: 'DELETE' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })

  const testChannel = useMutation({
    mutationFn: (body: TestChannelPayload) =>
      apiFetch<{ status: string }>('/api/v1/notifications/test', {
        credentials,
        method: 'POST',
        body: JSON.stringify(body),
      }),
  })

  return {
    channels: data ?? [],
    isLoading,
    error,
    createChannel,
    updateChannel,
    deleteChannel,
    testChannel,
  }
}
