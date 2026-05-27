import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Activity, Ban, Globe, Shield, Tag } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { StatusBadge } from '@/components/StatusBadge'
import { useAuth } from '@/contexts/AuthContext'
import { apiFetch, fetchInstanceStats, type Instance, type InstanceStats } from '@/lib/api'
import { useInstanceStatuses } from '@/hooks/useInstances'
import { useSyncStatus } from '@/hooks/useSyncStatus'

function StatRow({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-1.5 border-b last:border-0 border-border/50">
      <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {icon}
        {label}
      </span>
      <span className="text-sm font-medium tabular-nums">{value}</span>
    </div>
  )
}

function InstanceStatsCard({ inst, credentials }: { inst: Instance; credentials: Parameters<typeof fetchInstanceStats>[1] }) {
  const { statusMap, isLoaded: statusLoaded } = useInstanceStatuses(credentials)

  const { data: stats, isLoading, isError } = useQuery<InstanceStats>({
    queryKey: ['instance-stats', inst.id],
    queryFn: () => fetchInstanceStats(inst.id, credentials),
    refetchInterval: 60_000,
    staleTime: 55_000,
    retry: false,
  })

  const online = statusMap[inst.id]

  function fmtNum(n: number): string {
    if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
    if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
    return String(n)
  }

  function fmtMs(s: number): string {
    const ms = s * 1000
    if (ms < 1) return (ms * 1000).toFixed(0) + ' µs'
    return ms.toFixed(1) + ' ms'
  }

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <CardTitle className="text-base truncate">{inst.name}</CardTitle>
            <p className="text-xs text-muted-foreground truncate mt-0.5">{inst.address}</p>
            {stats?.version && (
              <span className="inline-flex items-center gap-1 mt-1 text-xs text-muted-foreground">
                <Tag className="h-3 w-3 shrink-0" />
                {stats.version}
              </span>
            )}
          </div>
          <div className="flex items-center gap-1.5 shrink-0 mt-0.5">
            <span
              className={`inline-block h-2 w-2 rounded-full ${
                !statusLoaded
                  ? 'bg-muted-foreground/30 animate-pulse'
                  : online
                  ? 'bg-green-500'
                  : 'bg-red-500'
              }`}
              title={!statusLoaded ? 'Checking…' : online ? 'Online' : 'Offline'}
            />
            <span className={`text-xs font-medium px-1.5 py-0.5 rounded-full ${
              inst.is_master
                ? 'bg-primary/10 text-primary'
                : 'bg-muted text-muted-foreground'
            }`}>
              {inst.is_master ? 'Master' : 'Slave'}
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-2 animate-pulse">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="h-6 rounded bg-muted" />
            ))}
          </div>
        )}
        {isError && (
          <p className="text-xs text-muted-foreground text-center py-3">Stats unavailable</p>
        )}
        {stats && (
          <div className="divide-y divide-border/50">
            <StatRow
              icon={<Globe className="h-3 w-3" />}
              label="Total DNS queries"
              value={fmtNum(stats.num_dns_queries)}
            />
            <StatRow
              icon={<Ban className="h-3 w-3" />}
              label="Blocked by filters"
              value={fmtNum(stats.num_blocked_filtering)}
            />
            <StatRow
              icon={<Shield className="h-3 w-3" />}
              label="Blocked malware/phishing"
              value={fmtNum(stats.num_replaced_safebrowsing)}
            />
            <StatRow
              icon={<Activity className="h-3 w-3" />}
              label="Avg processing time"
              value={fmtMs(stats.avg_processing_time)}
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}

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

      {instances && instances.length > 0 && (
        <div className="space-y-3">
          <h2 className="text-lg font-medium">Instances</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {instances.map((inst) => (
              <InstanceStatsCard key={inst.id} inst={inst} credentials={credentials} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
