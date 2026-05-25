import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { StatusBadge } from '@/components/StatusBadge'
import { useAuth } from '@/contexts/AuthContext'
import { apiFetch } from '@/lib/api'
import { useSyncStatus } from '@/hooks/useSyncStatus'
import { useSettings } from '@/hooks/useSettings'

export function Sync() {
  const { credentials } = useAuth()
  const qc = useQueryClient()
  const { current, last, status } = useSyncStatus(credentials)
  const { settings } = useSettings(credentials)
  const [cron, setCron] = useState('')
  const [savedCron, setSavedCron] = useState('')

  // Sync cron state from settings once loaded
  const currentCron = settings?.scheduler_cron ?? ''

  const runSync = useMutation({
    mutationFn: () =>
      apiFetch<{ run_id: string }>('/api/v1/sync/run', { credentials, method: 'POST' }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sync-status'] }),
  })

  const saveSchedule = useMutation({
    mutationFn: (expr: string) =>
      apiFetch('/api/v1/sync/schedule', {
        credentials,
        method: 'PUT',
        body: JSON.stringify({ cron: expr }),
      }),
    onSuccess: (_, expr) => {
      setSavedCron(expr)
      qc.invalidateQueries({ queryKey: ['settings'] })
    },
  })

  // Use controlled cron value: local edit > saved > from settings
  const displayCron = cron !== '' || savedCron !== '' ? cron : currentCron

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Sync</h1>

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Current Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <StatusBadge status={status} />
            {current && (
              <div className="text-sm text-muted-foreground space-y-0.5">
                <p>Run: {current.run_id.slice(0, 8)}…</p>
                <p>Triggered by: {current.triggered_by}</p>
                <p>Started: {new Date(current.started_at).toLocaleString()}</p>
              </div>
            )}
            {last && !current && (
              <div className="text-sm text-muted-foreground space-y-0.5">
                <p className="flex items-center gap-2">
                  Last: <StatusBadge status={last.status} />
                </p>
                <p>{new Date(last.started_at).toLocaleString()}</p>
              </div>
            )}
            <Button
              onClick={() => runSync.mutate()}
              disabled={runSync.isPending || status === 'running'}
            >
              {runSync.isPending ? 'Starting…' : 'Run Sync Now'}
            </Button>
            {runSync.isError && runSync.error.message === 'sync already in progress' && (
              <p className="text-sm text-destructive">Sync already in progress.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Schedule</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="cron">Cron Expression</Label>
              <Input
                id="cron"
                placeholder="0 */6 * * *"
                value={displayCron}
                onChange={(e) => setCron(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Leave blank to disable scheduled sync.
              </p>
            </div>
            <div className="flex gap-2">
              <Button
                onClick={() => saveSchedule.mutate(cron !== '' ? cron : displayCron)}
                disabled={saveSchedule.isPending}
              >
                {saveSchedule.isPending ? 'Saving…' : 'Save'}
              </Button>
              {currentCron !== '' && (
                <Button
                  variant="outline"
                  onClick={() => {
                    setCron('')
                    setSavedCron('')
                    saveSchedule.mutate('')
                  }}
                  disabled={saveSchedule.isPending}
                >
                  Clear
                </Button>
              )}
            </div>
            {saveSchedule.isError && (
              <p className="text-sm text-destructive">{saveSchedule.error.message}</p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
