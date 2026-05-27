export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export interface Credentials {
  username: string
  password: string
}

// Pre-encoded Basic auth value (base64). Used when credentials are restored
// from sessionStorage to avoid storing the raw password.
export interface EncodedCredentials {
  encodedAuth: string
}

export type AnyCredentials = Credentials | EncodedCredentials

function encodeCredentials(creds: AnyCredentials): string {
  if ('encodedAuth' in creds) return creds.encodedAuth
  return btoa(`${creds.username}:${creds.password}`)
}

export interface FetchOptions extends Omit<RequestInit, 'headers' | 'credentials'> {
  credentials: AnyCredentials | null
}

export async function apiFetch<T = unknown>(path: string, opts: FetchOptions): Promise<T> {
  const { credentials, ...rest } = opts
  const headers: Record<string, string> = {}
  headers['X-Requested-With'] = 'XMLHttpRequest'
  if (credentials) {
    headers['Authorization'] = 'Basic ' + encodeCredentials(credentials)
  }
  if (rest.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(path, { method: 'GET', ...rest, headers })
  if (!res.ok) {
    if (res.status === 401) {
      window.dispatchEvent(new Event('auth-clear'))
    }
    let message = res.statusText
    try {
      const body = await res.json() as { error?: string }
      if (body.error) message = body.error
    } catch {
      // ignore parse failure
    }
    throw new ApiError(res.status, message)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

// --- Domain types ---

export interface Instance {
  id: string
  name: string
  address: string
  username: string
  is_master: boolean
  tls_skip_verify: boolean
  created_at: string
  updated_at: string
}

export interface SyncConfigEntry {
  config_type: string
  enabled: boolean
}

export interface RunStatus {
  run_id: string
  triggered_by: string
  started_at: string
  finished_at?: string
  status: 'running' | 'success' | 'partial_failure' | 'error'
}

export interface SyncStatusResponse {
  current: RunStatus | null
  last: RunStatus | null
}

export interface Run {
  id: string
  triggered_by: string
  started_at: string
  finished_at?: string
  status: 'running' | 'success' | 'partial_failure' | 'error'
}

export interface RunResult {
  id: string
  run_id: string
  instance_id: string
  config_type: string
  status: 'success' | 'error'
  diff_json?: string
  error_msg?: string
  created_at: string
}

export interface RunDetail extends Run {
  results: RunResult[]
}

export interface Settings {
  ui_auth_enabled: boolean
  ui_username: string
  has_api_token: boolean
  scheduler_cron: string
  port: number
  ui_theme: string
}

export async function updateTheme(
  theme: 'dark' | 'light',
  credentials: AnyCredentials | null
): Promise<void> {
  await apiFetch<void>('/api/v1/settings/theme', {
    credentials,
    method: 'PUT',
    body: JSON.stringify({ theme }),
  })
}

export interface TestConnectionParams {
  address: string
  username: string
  password: string
  tls_skip_verify: boolean
}

export interface InstanceStats {
  version: string
  num_dns_queries: number
  num_blocked_filtering: number
  num_replaced_safebrowsing: number
  avg_processing_time: number
}

export async function fetchInstanceStats(id: string, credentials: AnyCredentials | null): Promise<InstanceStats> {
  return apiFetch<InstanceStats>(`/api/v1/instances/${id}/stats`, { credentials })
}

export interface InstanceStatus {
  id: string
  online: boolean
}

export async function fetchInstanceStatuses(credentials: AnyCredentials | null): Promise<InstanceStatus[]> {
  return apiFetch<InstanceStatus[]>('/api/v1/instances/statuses', { credentials })
}

export async function testConnection(
  params: TestConnectionParams,
  credentials: AnyCredentials | null
): Promise<void> {
  await apiFetch<{ ok: boolean }>('/api/v1/instances/test-connection', {
    credentials,
    method: 'POST',
    body: JSON.stringify(params),
  })
}

export async function exportBackup(credentials: AnyCredentials | null): Promise<void> {
  const headers: Record<string, string> = { 'X-Requested-With': 'XMLHttpRequest' }
  if (credentials) {
    headers['Authorization'] = 'Basic ' + encodeCredentials(credentials)
  }
  const res = await fetch('/api/v1/backup/export', { headers })
  if (!res.ok) throw new ApiError(res.status, res.statusText)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  const cd = res.headers.get('Content-Disposition') ?? ''
  const match = cd.match(/filename="([^"]+)"/)
  a.href = url
  a.download = match ? match[1] : 'aghsync-backup.json'
  a.click()
  URL.revokeObjectURL(url)
}

export async function importBackup(
  data: unknown,
  credentials: AnyCredentials | null
): Promise<void> {
  await apiFetch<void>('/api/v1/backup/restore', {
    credentials,
    method: 'POST',
    body: JSON.stringify(data),
  })
}
