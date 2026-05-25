import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { StatusBadge } from '@/components/StatusBadge'
import { useAuth } from '@/contexts/AuthContext'
import { useHistoryRun } from '@/hooks/useHistory'
import { useInstances } from '@/hooks/useInstances'
import { type RunResult } from '@/lib/api'

function duration(startedAt: string, finishedAt?: string): string {
  if (!finishedAt) return 'in progress'
  const ms = new Date(finishedAt).getTime() - new Date(startedAt).getTime()
  return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(1)}s`
}

interface DiffData {
  before: unknown
  after: unknown
}

function DiffViewer({ result }: { result: RunResult }) {
  const [expanded, setExpanded] = useState(false)

  if (!result.diff_json) return null

  let diff: DiffData
  try {
    diff = JSON.parse(result.diff_json) as DiffData
  } catch {
    return null
  }

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2 text-xs"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? <ChevronDown className="h-3 w-3 mr-1" /> : <ChevronRight className="h-3 w-3 mr-1" />}
        Diff
      </Button>
      {expanded && (
        <div className="mt-2 grid grid-cols-2 gap-2">
          <div>
            <p className="text-xs font-medium text-muted-foreground mb-1">Before</p>
            <pre className="text-xs bg-muted rounded p-2 overflow-auto max-h-48 whitespace-pre-wrap">
              {JSON.stringify(diff.before, null, 2)}
            </pre>
          </div>
          <div>
            <p className="text-xs font-medium text-muted-foreground mb-1">After</p>
            <pre className="text-xs bg-muted rounded p-2 overflow-auto max-h-48 whitespace-pre-wrap">
              {JSON.stringify(diff.after, null, 2)}
            </pre>
          </div>
        </div>
      )}
    </div>
  )
}

export function HistoryRun() {
  const { runId } = useParams<{ runId: string }>()
  const { credentials } = useAuth()
  const { run, isLoading } = useHistoryRun(credentials, runId ?? '')
  const { instances } = useInstances(credentials)

  const instanceName = (id: string) => instances.find((i) => i.id === id)?.name ?? id.slice(0, 8)

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>
  if (!run) return <p className="text-muted-foreground">Run not found.</p>

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/history" className="text-sm text-muted-foreground hover:underline">
          ← History
        </Link>
        <h1 className="text-2xl font-semibold">Run {run.id.slice(0, 8)}…</h1>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
        <div>
          <p className="text-muted-foreground">Status</p>
          <StatusBadge status={run.status} />
        </div>
        <div>
          <p className="text-muted-foreground">Triggered By</p>
          <p className="font-medium">{run.triggered_by}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Started</p>
          <p>{new Date(run.started_at).toLocaleString()}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Duration</p>
          <p>{duration(run.started_at, run.finished_at)}</p>
        </div>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Instance</TableHead>
            <TableHead>Config Type</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {run.results.map((result) => (
            <TableRow key={result.id}>
              <TableCell>{instanceName(result.instance_id)}</TableCell>
              <TableCell className="font-mono text-xs">{result.config_type}</TableCell>
              <TableCell>
                <StatusBadge status={result.status} />
              </TableCell>
              <TableCell>
                {result.error_msg ? (
                  <p className="text-xs text-destructive">{result.error_msg}</p>
                ) : (
                  <DiffViewer result={result} />
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
