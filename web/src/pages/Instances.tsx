import React, { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, ChevronDown, ChevronRight, Crown, Loader2, Pencil, Trash2, XCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { useAuth } from '@/contexts/AuthContext'
import { useInstances } from '@/hooks/useInstances'
import { ApiError, testConnection, type Instance, type SyncConfigEntry } from '@/lib/api'

const ALL_CONFIG_TYPES = [
  'blocked_services', 'clients', 'dhcp', 'dns',
  'filtering', 'parental', 'rewrite',
  'safebrowsing', 'safesearch', 'tls',
]

interface InstanceFormData {
  name: string
  address: string
  username: string
  password: string
  is_master: boolean
  tls_skip_verify: boolean
}

const EMPTY_FORM: InstanceFormData = {
  name: '',
  address: '',
  username: '',
  password: '',
  is_master: false,
  tls_skip_verify: false,
}

type TestStatus = 'idle' | 'testing' | 'ok' | 'failed'

export function Instances() {
  const { credentials } = useAuth()
  const qc = useQueryClient()
  const {
    instances,
    isLoading,
    createInstance,
    updateInstance,
    deleteInstance,
    promoteInstance,
    getSyncConfig,
    updateSyncConfig,
  } = useInstances(credentials)

  const [modalOpen, setModalOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Instance | null>(null)
  const [form, setForm] = useState<InstanceFormData>(EMPTY_FORM)
  const [confirmDelete, setConfirmDelete] = useState<Instance | null>(null)
  const [confirmPromote, setConfirmPromote] = useState<Instance | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [syncConfigs, setSyncConfigs] = useState<Record<string, SyncConfigEntry[]>>({})
  const debounceRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({})

  const [testStatus, setTestStatus] = useState<TestStatus>('idle')
  const [testError, setTestError] = useState('')
  const origConn = useRef({ address: '', username: '', tls_skip_verify: false })

  function openCreate() {
    setEditTarget(null)
    setForm(EMPTY_FORM)
    origConn.current = { address: '', username: '', tls_skip_verify: false }
    setTestStatus('idle')
    setTestError('')
    setModalOpen(true)
  }

  function openEdit(inst: Instance) {
    setEditTarget(inst)
    setForm({
      name: inst.name,
      address: inst.address,
      username: inst.username,
      password: '',
      is_master: inst.is_master,
      tls_skip_verify: inst.tls_skip_verify,
    })
    origConn.current = { address: inst.address, username: inst.username, tls_skip_verify: inst.tls_skip_verify }
    setTestStatus('ok')
    setTestError('')
    setModalOpen(true)
  }

  async function handleSubmit() {
    if (editTarget) {
      await updateInstance.mutateAsync({
        id: editTarget.id,
        name: form.name,
        address: form.address,
        username: form.username,
        password: form.password === '' ? null : form.password,
        tls_skip_verify: form.tls_skip_verify,
      })
    } else {
      await createInstance.mutateAsync(form)
    }
    setModalOpen(false)
  }

  async function handleTestConnection() {
    setTestStatus('testing')
    setTestError('')
    try {
      await testConnection(
        { address: form.address, username: form.username, password: form.password, tls_skip_verify: form.tls_skip_verify },
        credentials
      )
      setTestStatus('ok')
    } catch (e) {
      setTestStatus('failed')
      setTestError(e instanceof ApiError ? e.message : 'Connection failed')
    }
  }

  async function toggleExpand(id: string) {
    if (expandedId === id) {
      setExpandedId(null)
      return
    }
    setExpandedId(id)
    if (!syncConfigs[id]) {
      const cfg = await getSyncConfig(id)
      setSyncConfigs((prev) => ({ ...prev, [id]: cfg }))
    }
  }

  function handleConfigToggle(instanceId: string, configType: string, enabled: boolean) {
    setSyncConfigs((prev) => {
      const existing = prev[instanceId] ?? ALL_CONFIG_TYPES.map((ct) => ({ config_type: ct, enabled: true }))
      const updated = existing.map((e) =>
        e.config_type === configType ? { ...e, enabled } : e
      )
      clearTimeout(debounceRef.current[instanceId])
      debounceRef.current[instanceId] = setTimeout(() => {
        updateSyncConfig(instanceId, updated).then(() => {
          qc.invalidateQueries({ queryKey: ['sync-config', instanceId] })
        })
      }, 500)
      return { ...prev, [instanceId]: updated }
    })
  }

  // Cleanup debounce timers on unmount
  useEffect(() => {
    const ref = debounceRef.current
    return () => {
      Object.values(ref).forEach(clearTimeout)
    }
  }, [])

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Instances</h1>
        <Button onClick={openCreate}>Add Instance</Button>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            <TableHead />
            <TableHead>Name</TableHead>
            <TableHead>Address</TableHead>
            <TableHead>Role</TableHead>
            <TableHead>TLS Skip</TableHead>
            <TableHead>Created</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {instances.map((inst) => (
            <React.Fragment key={inst.id}>
              <TableRow>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => toggleExpand(inst.id)}
                    aria-label="Toggle sync config"
                  >
                    {expandedId === inst.id ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                  </Button>
                </TableCell>
                <TableCell className="font-medium">{inst.name}</TableCell>
                <TableCell className="text-muted-foreground">{inst.address}</TableCell>
                <TableCell>
                  {inst.is_master ? (
                    <Badge variant="default">Master</Badge>
                  ) : (
                    <Badge variant="secondary">Slave</Badge>
                  )}
                </TableCell>
                <TableCell>{inst.tls_skip_verify ? 'Yes' : 'No'}</TableCell>
                <TableCell className="text-muted-foreground">
                  {new Date(inst.created_at).toLocaleDateString()}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => openEdit(inst)}
                      aria-label="Edit"
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    {!inst.is_master && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setConfirmPromote(inst)}
                        aria-label="Promote to master"
                      >
                        <Crown className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setConfirmDelete(inst)}
                      aria-label="Delete"
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>

              {expandedId === inst.id && (
                <TableRow key={`${inst.id}-config`}>
                  <TableCell colSpan={7} className="bg-muted/30 px-8 py-4">
                    <p className="text-sm font-medium mb-3">Sync Config</p>
                    <div className="grid grid-cols-3 gap-2">
                      {(
                        syncConfigs[inst.id] ??
                        ALL_CONFIG_TYPES.map((ct) => ({ config_type: ct, enabled: true }))
                      ).map((entry) => (
                        <label
                          key={entry.config_type}
                          className="flex items-center gap-2 cursor-pointer"
                        >
                          <Checkbox
                            checked={entry.enabled}
                            onCheckedChange={(v) =>
                              handleConfigToggle(inst.id, entry.config_type, v === true)
                            }
                          />
                          <span className="text-sm">{entry.config_type}</span>
                        </label>
                      ))}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </React.Fragment>
          ))}
        </TableBody>
      </Table>

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editTarget ? 'Edit Instance' : 'Add Instance'}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="address">Address</Label>
              <Input
                id="address"
                value={form.address}
                onChange={(e) => {
                  const v = e.target.value
                  setForm((f) => ({ ...f, address: v }))
                  if (v !== origConn.current.address) setTestStatus('idle')
                }}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={form.username}
                onChange={(e) => {
                  const v = e.target.value
                  setForm((f) => ({ ...f, username: v }))
                  if (v !== origConn.current.username) setTestStatus('idle')
                }}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="password">
                Password {editTarget ? '(leave blank to keep existing)' : ''}
              </Label>
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={(e) => {
                  const v = e.target.value
                  setForm((f) => ({ ...f, password: v }))
                  if (v !== '') setTestStatus('idle')
                }}
              />
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={form.is_master}
                onCheckedChange={(v) => setForm((f) => ({ ...f, is_master: v === true }))}
                disabled={!!editTarget}
              />
              <span className="text-sm">Is Master</span>
            </label>
            <label className="flex items-center gap-2 cursor-pointer">
              <Checkbox
                checked={form.tls_skip_verify}
                onCheckedChange={(v) => {
                  const checked = v === true
                  setForm((f) => ({ ...f, tls_skip_verify: checked }))
                  if (checked !== origConn.current.tls_skip_verify) setTestStatus('idle')
                }}
              />
              <span className="text-sm">TLS Skip Verify</span>
            </label>
            <div className="flex items-center gap-3 pt-1">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={handleTestConnection}
                disabled={testStatus === 'testing' || !form.address.trim()}
              >
                {testStatus === 'testing' ? (
                  <><Loader2 className="h-4 w-4 animate-spin mr-1" />Testing…</>
                ) : (
                  'Test Connection'
                )}
              </Button>
              {testStatus === 'ok' && (
                <span className="flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                  <CheckCircle2 className="h-4 w-4" />Connected
                </span>
              )}
              {testStatus === 'failed' && (
                <span className="flex items-center gap-1 text-sm text-destructive max-w-xs truncate" title={testError}>
                  <XCircle className="h-4 w-4 shrink-0" />{testError}
                </span>
              )}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setModalOpen(false)}>Cancel</Button>
            <Button
              onClick={handleSubmit}
              disabled={createInstance.isPending || updateInstance.isPending || testStatus !== 'ok'}
            >
              {editTarget ? 'Save' : 'Add'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!confirmDelete}
        onOpenChange={(o) => !o && setConfirmDelete(null)}
        title="Delete Instance"
        description={`Delete "${confirmDelete?.name}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={() => {
          if (confirmDelete) deleteInstance.mutate(confirmDelete.id)
          setConfirmDelete(null)
        }}
        loading={deleteInstance.isPending}
      />

      <ConfirmDialog
        open={!!confirmPromote}
        onOpenChange={(o) => !o && setConfirmPromote(null)}
        title="Promote Instance"
        description={`Promote "${confirmPromote?.name}" to master? This will demote the current master to slave.`}
        confirmLabel="Promote"
        onConfirm={() => {
          if (confirmPromote) promoteInstance.mutate(confirmPromote.id)
          setConfirmPromote(null)
        }}
        loading={promoteInstance.isPending}
      />
    </div>
  )
}
