import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { StatusBadge } from '@/components/StatusBadge'
import { useAuth } from '@/contexts/AuthContext'
import { useHistory } from '@/hooks/useHistory'

const PAGE_SIZE = 20

function duration(startedAt: string, finishedAt?: string): string {
  if (!finishedAt) return '–'
  const ms = new Date(finishedAt).getTime() - new Date(startedAt).getTime()
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export function History() {
  const { credentials } = useAuth()
  const navigate = useNavigate()
  const [offset, setOffset] = useState(0)
  const { runs, isLoading } = useHistory(credentials, PAGE_SIZE, offset)

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">History</h1>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Run ID</TableHead>
            <TableHead>Triggered By</TableHead>
            <TableHead>Started At</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>Status</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {runs.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="text-center text-muted-foreground">
                No runs yet.
              </TableCell>
            </TableRow>
          ) : (
            runs.map((run) => (
              <TableRow
                key={run.id}
                className="cursor-pointer"
                onClick={() => navigate(`/history/${run.id}`)}
              >
                <TableCell className="font-mono text-xs">{run.id.slice(0, 8)}…</TableCell>
                <TableCell>{run.triggered_by}</TableCell>
                <TableCell className="text-muted-foreground">
                  {new Date(run.started_at).toLocaleString()}
                </TableCell>
                <TableCell>{duration(run.started_at, run.finished_at)}</TableCell>
                <TableCell>
                  <StatusBadge status={run.status} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={offset === 0}
          onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={runs.length < PAGE_SIZE}
          onClick={() => setOffset((o) => o + PAGE_SIZE)}
        >
          Next
        </Button>
      </div>
    </div>
  )
}
