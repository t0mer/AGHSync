import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { StatusBadge } from '@/components/StatusBadge'
import { useAuth } from '@/contexts/AuthContext'
import { apiFetch, type Instance } from '@/lib/api'
import { useSyncStatus } from '@/hooks/useSyncStatus'

export function Dashboard() {
  const { credentials } = useAuth()
  const qc = useQueryClient()

  const { data: instances } = useQuery<Instance[]>({
    queryKey: ['instances'],
    queryFn: () => apiFetch<Instance[]>('/api/v1/instances', { credentials }),
  })

  const { current, last, status } = useSyncStatus(credentials)

  const runSync = useMutation({
    mutationFn: () =>
      apiFetch<{ run_id: string }>('/api/v1/sync/run', { credentials, method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sync-status'] }),
    onError: (err: Error) => {
      if (err.message === 'sync already in progress') return
    },
  })

  const master = instances?.find((i) => i.is_master)

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Dashboard</h1>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Master Instance</CardTitle>
          </CardHeader>
          <CardContent>
            {master ? (
              <div className="space-y-1">
                <p className="font-medium">{master.name}</p>
                <p className="text-sm text-muted-foreground">{master.address}</p>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">No master configured.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Sync Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <StatusBadge status={status} />
            {current && (
              <div className="text-xs text-muted-foreground space-y-0.5">
                <p>Run: {current.run_id.slice(0, 8)}…</p>
                <p>Triggered by: {current.triggered_by}</p>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Last Run</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {last ? (
              <>
                <StatusBadge status={last.status} />
                <div className="text-xs text-muted-foreground space-y-0.5">
                  <p>{new Date(last.started_at).toLocaleString()}</p>
                  <p>Triggered by: {last.triggered_by}</p>
                </div>
                <Link
                  to={`/history/${last.run_id}`}
                  className="text-xs text-primary underline-offset-4 hover:underline"
                >
                  View details →
                </Link>
              </>
            ) : (
              <p className="text-sm text-muted-foreground">No runs yet.</p>
            )}
          </CardContent>
        </Card>
      </div>

      <Button
        onClick={() => runSync.mutate()}
        disabled={runSync.isPending || status === 'running'}
      >
        {runSync.isPending ? 'Starting…' : 'Run Sync Now'}
      </Button>
      {runSync.isError && runSync.error.message === 'sync already in progress' && (
        <p className="text-sm text-destructive">Sync already in progress.</p>
      )}
    </div>
  )
}
