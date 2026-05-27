import { useState } from 'react'
import { Bell, CheckCircle2, Loader2, Pencil, Trash2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { useAuth } from '@/contexts/AuthContext'
import { useNotifications } from '@/hooks/useNotifications'
import { ApiError, type NotificationChannel, type NotificationChannelType } from '@/lib/api'

interface FormData {
  name: string
  type: NotificationChannelType
  config: string
  notify_success: boolean
  notify_failure: boolean
  enabled: boolean
}

const EMPTY_FORM: FormData = {
  name: '',
  type: 'shoutrrr',
  config: '',
  notify_success: true,
  notify_failure: true,
  enabled: true,
}

const TYPE_LABELS: Record<NotificationChannelType, string> = {
  shoutrrr: 'Shoutrrr',
  greenapi: 'GreenAPI (WhatsApp)',
  whatsapp: 'WhatsApp Web',
}

const CONFIG_PLACEHOLDERS: Record<NotificationChannelType, string> = {
  shoutrrr: '{"url":"slack://token@channel"}',
  greenapi: '{"instance_id":"1234","api_token":"token","phone":"1631601XXXX"}',
  whatsapp: '{"api_url":"http://localhost:3000","phone":"1631601XXXX"}',
}

const CONFIG_HINTS: Record<NotificationChannelType, string> = {
  shoutrrr: 'Shoutrrr URL — supports Slack, Discord, Telegram, Gotify, SMTP, and more. See shoutrrr.containerize.it for the URL format.',
  greenapi: 'GreenAPI cloud WhatsApp service. Get your Instance ID and API token from console.green-api.com.',
  whatsapp: 'Self-hosted go-whatsapp-web-multidevice instance. Set the base URL and recipient phone number.',
}

type TestState = 'idle' | 'testing' | 'ok' | 'failed'

export function Notifications() {
  const { credentials } = useAuth()
  const { channels, isLoading, createChannel, updateChannel, deleteChannel, testChannel } =
    useNotifications(credentials)

  const [modalOpen, setModalOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<NotificationChannel | null>(null)
  const [form, setForm] = useState<FormData>(EMPTY_FORM)
  const [confirmDelete, setConfirmDelete] = useState<NotificationChannel | null>(null)
  const [submitError, setSubmitError] = useState('')
  const [testState, setTestState] = useState<TestState>('idle')
  const [testError, setTestError] = useState('')

  function openCreate() {
    setEditTarget(null)
    setForm(EMPTY_FORM)
    setSubmitError('')
    setTestState('idle')
    setTestError('')
    setModalOpen(true)
  }

  function openEdit(ch: NotificationChannel) {
    setEditTarget(ch)
    setForm({
      name: ch.name,
      type: ch.type,
      config: ch.config,
      notify_success: ch.notify_success,
      notify_failure: ch.notify_failure,
      enabled: ch.enabled,
    })
    setSubmitError('')
    setTestState('idle')
    setTestError('')
    setModalOpen(true)
  }

  async function handleTest() {
    setTestState('testing')
    setTestError('')
    try {
      await testChannel.mutateAsync({ type: form.type, config: form.config })
      setTestState('ok')
    } catch (e) {
      setTestState('failed')
      setTestError(e instanceof ApiError ? e.message : 'Test failed')
    }
  }

  async function handleSubmit() {
    setSubmitError('')
    try {
      if (editTarget) {
        await updateChannel.mutateAsync({ id: editTarget.id, ...form })
      } else {
        await createChannel.mutateAsync(form)
      }
      setModalOpen(false)
    } catch (e) {
      setSubmitError(e instanceof ApiError ? e.message : 'Failed to save channel')
    }
  }

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Notifications</h1>
        <Button onClick={openCreate}>Add Channel</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Bell className="h-4 w-4" />
            Notification Channels
          </CardTitle>
          <p className="text-xs text-muted-foreground mt-1">
            Channels receive a message after every sync run, with a summary of what changed.
            Configure each channel to fire on success, failure, or both.
          </p>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>On Success</TableHead>
                <TableHead>On Failure</TableHead>
                <TableHead>Status</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    No notification channels configured.
                  </TableCell>
                </TableRow>
              ) : (
                channels.map((ch) => (
                  <TableRow key={ch.id}>
                    <TableCell className="font-medium">{ch.name}</TableCell>
                    <TableCell>{TYPE_LABELS[ch.type]}</TableCell>
                    <TableCell>
                      {ch.notify_success ? (
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-muted-foreground" />
                      )}
                    </TableCell>
                    <TableCell>
                      {ch.notify_failure ? (
                        <CheckCircle2 className="h-4 w-4 text-green-500" />
                      ) : (
                        <XCircle className="h-4 w-4 text-muted-foreground" />
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={ch.enabled ? 'default' : 'secondary'}>
                        {ch.enabled ? 'Active' : 'Inactive'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">
                        <Button variant="ghost" size="icon" onClick={() => openEdit(ch)} aria-label="Edit">
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => setConfirmDelete(ch)}
                          aria-label="Delete"
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Add / Edit dialog */}
      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{editTarget ? 'Edit Channel' : 'Add Channel'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {/* Name */}
            <div className="space-y-1.5">
              <Label htmlFor="ch-name">Name</Label>
              <Input
                id="ch-name"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>

            {/* Type */}
            <div className="space-y-1.5">
              <Label htmlFor="ch-type">Type</Label>
              <select
                id="ch-type"
                className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={form.type}
                onChange={(e) => {
                  const t = e.target.value as NotificationChannelType
                  setForm((f) => ({ ...f, type: t, config: '' }))
                  setTestState('idle')
                }}
              >
                {(Object.entries(TYPE_LABELS) as [NotificationChannelType, string][]).map(([val, label]) => (
                  <option key={val} value={val}>{label}</option>
                ))}
              </select>
            </div>

            {/* Config */}
            <div className="space-y-1.5">
              <Label htmlFor="ch-config">Config (JSON)</Label>
              <textarea
                id="ch-config"
                rows={3}
                className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring font-mono"
                placeholder={CONFIG_PLACEHOLDERS[form.type]}
                value={form.config}
                onChange={(e) => {
                  setForm((f) => ({ ...f, config: e.target.value }))
                  setTestState('idle')
                }}
              />
              <p className="text-xs text-muted-foreground">{CONFIG_HINTS[form.type]}</p>
            </div>

            {/* Test connection */}
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleTest}
                disabled={testState === 'testing' || !form.config.trim()}
              >
                {testState === 'testing' ? (
                  <><Loader2 className="h-4 w-4 animate-spin mr-1" />Testing…</>
                ) : 'Send Test'}
              </Button>
              {testState === 'ok' && (
                <span className="flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                  <CheckCircle2 className="h-4 w-4" />Message sent
                </span>
              )}
              {testState === 'failed' && (
                <span className="flex items-center gap-1 text-sm text-destructive max-w-xs truncate" title={testError}>
                  <XCircle className="h-4 w-4 shrink-0" />{testError}
                </span>
              )}
            </div>

            {/* Toggles */}
            <div className="space-y-2 pt-1">
              <label className="flex items-center gap-2 cursor-pointer">
                <Checkbox
                  checked={form.notify_success}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, notify_success: v === true }))}
                />
                <span className="text-sm">Notify on success</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <Checkbox
                  checked={form.notify_failure}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, notify_failure: v === true }))}
                />
                <span className="text-sm">Notify on failure / partial failure</span>
              </label>
              <div className="flex items-center gap-2 pt-1">
                <Switch
                  id="ch-enabled"
                  checked={form.enabled}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, enabled: v }))}
                />
                <Label htmlFor="ch-enabled">Enabled</Label>
              </div>
            </div>
          </div>

          {submitError && <p className="text-sm text-destructive -mt-1">{submitError}</p>}

          <DialogFooter>
            <Button variant="outline" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button
              onClick={handleSubmit}
              disabled={createChannel.isPending || updateChannel.isPending}
            >
              {editTarget ? 'Save' : 'Add'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={(o) => !o && setConfirmDelete(null)}
        title="Delete Channel"
        description={`Delete "${confirmDelete?.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={() => {
          if (confirmDelete) deleteChannel.mutate(confirmDelete.id)
          setConfirmDelete(null)
        }}
        loading={deleteChannel.isPending}
      />
    </div>
  )
}
