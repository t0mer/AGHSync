import { useMemo, useState } from 'react'
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

// --- diff engine ---

type DiffLine = { text: string; type: 'equal' | 'added' | 'removed' }

function computeDiff(before: string, after: string): DiffLine[] {
  const a = before.split('\n')
  const b = after.split('\n')
  const m = a.length
  const n = b.length

  // LCS DP table
  const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = a[i - 1] === b[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }

  // Backtrack
  const result: DiffLine[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && a[i - 1] === b[j - 1]) {
      result.unshift({ text: a[i - 1], type: 'equal' })
      i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      result.unshift({ text: b[j - 1], type: 'added' })
      j--
    } else {
      result.unshift({ text: a[i - 1], type: 'removed' })
      i--
    }
  }
  return result
}

// --- DiffViewer ---

function DiffViewer({ result }: { result: RunResult }) {
  const [expanded, setExpanded] = useState(false)

  const diff = useMemo<DiffLine[] | null>(() => {
    if (!result.diff_json) return null
    try {
      const { before, after } = JSON.parse(result.diff_json) as { before: unknown; after: unknown }
      return computeDiff(
        JSON.stringify(before, null, 2),
        JSON.stringify(after, null, 2),
      )
    } catch {
      return null
    }
  }, [result.diff_json])

  if (!diff) return null

  const added = diff.filter((l) => l.type === 'added').length
  const removed = diff.filter((l) => l.type === 'removed').length
  const hasChanges = added > 0 || removed > 0

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2 text-xs gap-1"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        Diff
        {hasChanges && (
          <span className="ml-1 font-mono">
            {added > 0 && <span className="text-green-600 dark:text-green-400">+{added}</span>}
            {added > 0 && removed > 0 && <span className="text-muted-foreground mx-0.5">/</span>}
            {removed > 0 && <span className="text-red-500">-{removed}</span>}
          </span>
        )}
      </Button>

      {expanded && (
        <div className="mt-2 rounded border border-border overflow-hidden">
          {!hasChanges ? (
            <p className="text-xs text-muted-foreground px-3 py-2">No changes</p>
          ) : (
            <div className="overflow-auto max-h-96">
              <table className="w-full text-xs font-mono border-collapse">
                <tbody>
                  {diff.map((line, idx) => {
                    if (line.type === 'equal') {
                      return (
                        <tr key={idx}>
                          <td className="select-none w-4 px-2 text-muted-foreground/40 text-right border-r border-border bg-background"> </td>
                          <td className="px-3 py-px whitespace-pre bg-background text-foreground">{line.text}</td>
                        </tr>
                      )
                    }
                    if (line.type === 'added') {
                      return (
                        <tr key={idx} className="bg-green-500/10 dark:bg-green-500/15">
                          <td className="select-none w-4 px-2 text-green-600 dark:text-green-400 text-right border-r border-green-500/20 font-bold">+</td>
                          <td className="px-3 py-px whitespace-pre text-green-800 dark:text-green-300">{line.text}</td>
                        </tr>
                      )
                    }
                    // removed
                    return (
                      <tr key={idx} className="bg-red-500/10 dark:bg-red-500/15">
                        <td className="select-none w-4 px-2 text-red-500 text-right border-r border-red-500/20 font-bold">-</td>
                        <td className="px-3 py-px whitespace-pre text-red-700 dark:text-red-400 line-through decoration-red-400/60">{line.text}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
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
