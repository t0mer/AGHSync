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

// Per-provider field state
interface ShoutrrrFields { url: string }
interface GreenAPIFields { instance_id: string; token: string; recipient: string; api_url: string }
interface WhatsAppFields { base_url: string; recipient: string; username: string; password: string }

interface FormData {
  name: string
  type: NotificationChannelType
  notify_success: boolean
  notify_failure: boolean
  enabled: boolean
  shoutrrr: ShoutrrrFields
  greenapi: GreenAPIFields
  whatsapp: WhatsAppFields
}

const EMPTY_FORM: FormData = {
  name: '',
  type: 'shoutrrr',
  notify_success: true,
  notify_failure: true,
  enabled: true,
  shoutrrr: { url: '' },
  greenapi: { instance_id: '', token: '', recipient: '', api_url: '' },
  whatsapp: { base_url: '', recipient: '', username: '', password: '' },
}

function buildConfig(form: FormData): string {
  switch (form.type) {
    case 'shoutrrr': return JSON.stringify(form.shoutrrr)
    case 'greenapi': return JSON.stringify(form.greenapi)
    case 'whatsapp': return JSON.stringify(form.whatsapp)
  }
}

function parseIntoForm(base: FormData, type: NotificationChannelType, configJson: string): FormData {
  try {
    const parsed = JSON.parse(configJson)
    switch (type) {
      case 'shoutrrr': return { ...base, type, shoutrrr: { url: parsed.url ?? '' } }
      case 'greenapi': return { ...base, type, greenapi: { instance_id: parsed.instance_id ?? '', token: parsed.token ?? '', recipient: parsed.recipient ?? '', api_url: parsed.api_url ?? '' } }
      case 'whatsapp': return { ...base, type, whatsapp: { base_url: parsed.base_url ?? '', recipient: parsed.recipient ?? '', username: parsed.username ?? '', password: parsed.password ?? '' } }
    }
  } catch {
    return { ...base, type }
  }
}

const TYPE_LABELS: Record<NotificationChannelType, string> = {
  shoutrrr: 'Shoutrrr',
  greenapi: 'GreenAPI (WhatsApp)',
  whatsapp: 'WhatsApp Web',
}

type TestState = 'idle' | 'testing' | 'ok' | 'failed'

// Shared input field row
function Field({ id, label, value, onChange, type = 'text', placeholder, hint }: {
  id: string; label: string; value: string
  onChange: (v: string) => void
  type?: string; placeholder?: string; hint?: string
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} type={type} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} />
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

function ShoutrrrForm({ fields, onChange }: { fields: ShoutrrrFields; onChange: (f: ShoutrrrFields) => void }) {
  return (
    <Field
      id="sh-url" label="Shoutrrr URL" value={fields.url}
      onChange={(v) => onChange({ url: v })}
      placeholder="slack://token@channel"
      hint="Supports Slack, Discord, Telegram, Gotify, SMTP, and more. See shoutrrr.containerize.it for URL formats."
    />
  )
}

function GreenAPIForm({ fields, onChange }: { fields: GreenAPIFields; onChange: (f: GreenAPIFields) => void }) {
  return (
    <div className="space-y-3">
      <Field id="ga-instance" label="Instance ID" value={fields.instance_id}
        onChange={(v) => onChange({ ...fields, instance_id: v })}
        placeholder="7103251345" hint="Found in your GreenAPI console (console.green-api.com)." />
      <Field id="ga-token" label="Token" value={fields.token}
        onChange={(v) => onChange({ ...fields, token: v })}
        type="password" placeholder="••••••••" />
      <Field id="ga-recipient" label="Recipient Phone" value={fields.recipient}
        onChange={(v) => onChange({ ...fields, recipient: v })}
        placeholder="972502961865" hint="International format without + or spaces (country code + number)." />
      <Field id="ga-apiurl" label="API URL (optional)" value={fields.api_url}
        onChange={(v) => onChange({ ...fields, api_url: v })}
        placeholder="https://7103.api.greenapi.com"
        hint="Leave blank to use the default. Set to the cluster URL shown in your GreenAPI console." />
    </div>
  )
}

function WhatsAppForm({ fields, onChange }: { fields: WhatsAppFields; onChange: (f: WhatsAppFields) => void }) {
  return (
    <div className="space-y-3">
      <Field id="wa-url" label="Base URL" value={fields.base_url}
        onChange={(v) => onChange({ ...fields, base_url: v })}
        placeholder="http://localhost:3000"
        hint="Base URL of your go-whatsapp-web-multidevice instance." />
      <Field id="wa-recipient" label="Recipient Phone" value={fields.recipient}
        onChange={(v) => onChange({ ...fields, recipient: v })}
        placeholder="972502961865" hint="International format without + or spaces (country code + number)." />
      <Field id="wa-user" label="Username (optional)" value={fields.username}
        onChange={(v) => onChange({ ...fields, username: v })} placeholder="admin" />
      <Field id="wa-pass" label="Password (optional)" value={fields.password}
        onChange={(v) => onChange({ ...fields, password: v })} type="password" placeholder="••••••••" />
    </div>
  )
}

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
    setForm(parseIntoForm({ ...EMPTY_FORM, name: ch.name, notify_success: ch.notify_success, notify_failure: ch.notify_failure, enabled: ch.enabled }, ch.type, ch.config))
    setSubmitError('')
    setTestState('idle')
    setTestError('')
    setModalOpen(true)
  }

  function resetTestState() { setTestState('idle'); setTestError('') }

  async function handleTest() {
    setTestState('testing')
    setTestError('')
    try {
      await testChannel.mutateAsync({ type: form.type, config: buildConfig(form) })
      setTestState('ok')
    } catch (e) {
      setTestState('failed')
      setTestError(e instanceof ApiError ? e.message : 'Test failed')
    }
  }

  async function handleSubmit() {
    setSubmitError('')
    const payload = { name: form.name, type: form.type, config: buildConfig(form), notify_success: form.notify_success, notify_failure: form.notify_failure, enabled: form.enabled }
    try {
      if (editTarget) {
        await updateChannel.mutateAsync({ id: editTarget.id, ...payload })
      } else {
        await createChannel.mutateAsync(payload)
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
            Channels receive a message after every sync run with a summary of what changed.
            Configure each to fire on success, failure, or both.
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
                      {ch.notify_success
                        ? <CheckCircle2 className="h-4 w-4 text-green-500" />
                        : <XCircle className="h-4 w-4 text-muted-foreground" />}
                    </TableCell>
                    <TableCell>
                      {ch.notify_failure
                        ? <CheckCircle2 className="h-4 w-4 text-green-500" />
                        : <XCircle className="h-4 w-4 text-muted-foreground" />}
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
                        <Button variant="ghost" size="icon" onClick={() => setConfirmDelete(ch)} aria-label="Delete">
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
            <Field id="ch-name" label="Name" value={form.name}
              onChange={(v) => setForm((f) => ({ ...f, name: v }))} />

            {/* Type */}
            <div className="space-y-1.5">
              <Label htmlFor="ch-type">Type</Label>
              <select
                id="ch-type"
                className="flex h-9 w-full rounded-md border border-input bg-background text-foreground px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                value={form.type}
                onChange={(e) => {
                  setForm((f) => ({ ...f, type: e.target.value as NotificationChannelType }))
                  resetTestState()
                }}
              >
                {(Object.entries(TYPE_LABELS) as [NotificationChannelType, string][]).map(([val, label]) => (
                  <option key={val} value={val}>{label}</option>
                ))}
              </select>
            </div>

            {/* Provider-specific fields */}
            {form.type === 'shoutrrr' && (
              <ShoutrrrForm fields={form.shoutrrr}
                onChange={(v) => { setForm((f) => ({ ...f, shoutrrr: v })); resetTestState() }} />
            )}
            {form.type === 'greenapi' && (
              <GreenAPIForm fields={form.greenapi}
                onChange={(v) => { setForm((f) => ({ ...f, greenapi: v })); resetTestState() }} />
            )}
            {form.type === 'whatsapp' && (
              <WhatsAppForm fields={form.whatsapp}
                onChange={(v) => { setForm((f) => ({ ...f, whatsapp: v })); resetTestState() }} />
            )}

            {/* Send Test */}
            <div className="flex items-center gap-3">
              <Button type="button" variant="outline" size="sm" onClick={handleTest}
                disabled={testState === 'testing'}>
                {testState === 'testing'
                  ? <><Loader2 className="h-4 w-4 animate-spin mr-1" />Testing…</>
                  : 'Send Test'}
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
                <Checkbox checked={form.notify_success}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, notify_success: v === true }))} />
                <span className="text-sm">Notify on success</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <Checkbox checked={form.notify_failure}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, notify_failure: v === true }))} />
                <span className="text-sm">Notify on failure / partial failure</span>
              </label>
              <div className="flex items-center gap-2 pt-1">
                <Switch id="ch-enabled" checked={form.enabled}
                  onCheckedChange={(v) => setForm((f) => ({ ...f, enabled: v }))} />
                <Label htmlFor="ch-enabled">Enabled</Label>
              </div>
            </div>
          </div>

          {submitError && <p className="text-sm text-destructive -mt-1">{submitError}</p>}

          <DialogFooter>
            <Button variant="outline" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button onClick={handleSubmit} disabled={createChannel.isPending || updateChannel.isPending}>
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
