import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiFetch, fetchInstanceStatuses, type AnyCredentials, type Instance, type InstanceStatus, type SyncConfigEntry } from '@/lib/api'

export function useInstances(credentials: AnyCredentials | null) {
  const qc = useQueryClient()

  const { data, isLoading, error } = useQuery<Instance[]>({
    queryKey: ['instances'],
    queryFn: () => apiFetch<Instance[]>('/api/v1/instances', { credentials }),
  })

  function invalidateAll() {
    qc.invalidateQueries({ queryKey: ['instances'] })
    qc.invalidateQueries({ queryKey: ['instance-statuses'] })
  }

  const createInstance = useMutation({
    mutationFn: (body: {
      name: string
      address: string
      username: string
      password: string
      is_master: boolean
      tls_skip_verify: boolean
    }) =>
      apiFetch<Instance>('/api/v1/instances', {
        credentials,
        method: 'POST',
        body: JSON.stringify(body),
      }),
    onSuccess: invalidateAll,
  })

  const updateInstance = useMutation({
    mutationFn: ({
      id,
      ...body
    }: {
      id: string
      name: string
      address: string
      username: string
      password: string | null
      tls_skip_verify: boolean
    }) =>
      apiFetch<Instance>(`/api/v1/instances/${id}`, {
        credentials,
        method: 'PUT',
        body: JSON.stringify(body),
      }),
    onSuccess: invalidateAll,
  })

  const deleteInstance = useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/v1/instances/${id}`, { credentials, method: 'DELETE' }),
    onSuccess: invalidateAll,
  })

  const promoteInstance = useMutation({
    mutationFn: (id: string) =>
      apiFetch<Instance>(`/api/v1/instances/${id}/promote`, { credentials, method: 'PUT' }),
    onSuccess: invalidateAll,
  })

  const setSyncEnabled = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      apiFetch<Instance>(`/api/v1/instances/${id}/sync-enabled`, {
        credentials,
        method: 'PUT',
        body: JSON.stringify({ enabled }),
      }),
    onSuccess: invalidateAll,
  })

  const getSyncConfig = (id: string) =>
    apiFetch<SyncConfigEntry[]>(`/api/v1/instances/${id}/sync-config`, { credentials })

  const updateSyncConfig = (id: string, config: SyncConfigEntry[]) =>
    apiFetch<SyncConfigEntry[]>(`/api/v1/instances/${id}/sync-config`, {
      credentials,
      method: 'PUT',
      body: JSON.stringify({ config }),
    })

  return {
    instances: data ?? [],
    isLoading,
    error,
    createInstance,
    updateInstance,
    deleteInstance,
    promoteInstance,
    setSyncEnabled,
    getSyncConfig,
    updateSyncConfig,
  }
}

export function useInstanceStatuses(credentials: AnyCredentials | null) {
  const { data } = useQuery<InstanceStatus[]>({
    queryKey: ['instance-statuses'],
    queryFn: () => fetchInstanceStatuses(credentials),
    refetchInterval: 60_000,
    staleTime: 55_000,
  })

  const statusMap: Record<string, boolean | undefined> = {}
  for (const s of data ?? []) {
    statusMap[s.id] = s.online
  }
  return { statusMap, isLoaded: data !== undefined }
}
