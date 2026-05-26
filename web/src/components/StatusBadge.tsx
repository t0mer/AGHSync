import { Badge } from '@/components/ui/badge'

type RunStatus = 'running' | 'success' | 'partial_failure' | 'error' | 'idle'

const LABEL: Record<RunStatus, string> = {
  running: 'Running',
  success: 'Success',
  partial_failure: 'Partial Failure',
  error: 'Error',
  idle: 'Idle',
}

const VARIANT: Record<RunStatus, 'running' | 'success' | 'destructive' | 'warning' | 'secondary'> = {
  running: 'running',
  success: 'success',
  partial_failure: 'warning',
  error: 'destructive',
  idle: 'secondary',
}

export function StatusBadge({ status }: { status: RunStatus }) {
  return <Badge variant={VARIANT[status]}>{LABEL[status]}</Badge>
}
