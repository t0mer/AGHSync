import { useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/ConfirmDialog'
import { useAuth } from '@/contexts/AuthContext'
import { useSettings } from '@/hooks/useSettings'
import { exportBackup, importBackup } from '@/lib/api'

export function Settings() {
  const { credentials } = useAuth()
  const { settings, isLoading, updateUIAuth, generateToken, deleteToken } = useSettings(credentials)

  const [authUsername, setAuthUsername] = useState('')
  const [authPassword, setAuthPassword] = useState('')

  const [tokenModalOpen, setTokenModalOpen] = useState(false)
  const [newToken, setNewToken] = useState('')
  const [copied, setCopied] = useState(false)
  const [confirmDeleteToken, setConfirmDeleteToken] = useState(false)

  const [exportError, setExportError] = useState('')
  const [exportBusy, setExportBusy] = useState(false)
  const [importError, setImportError] = useState('')
  const [importBusy, setImportBusy] = useState(false)
  const [confirmRestore, setConfirmRestore] = useState(false)
  const [pendingBackup, setPendingBackup] = useState<unknown>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  async function handleEnableAuth() {
    await updateUIAuth.mutateAsync({
      enabled: true,
      username: authUsername,
      password: authPassword,
    })
    setAuthUsername('')
    setAuthPassword('')
  }

  async function handleDisableAuth() {
    await updateUIAuth.mutateAsync({ enabled: false })
  }

  async function handleGenerateToken() {
    const res = await generateToken.mutateAsync()
    setNewToken(res.token)
    setTokenModalOpen(true)
  }

  function copyToken() {
    navigator.clipboard.writeText(newToken).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  async function handleExport() {
    setExportError('')
    setExportBusy(true)
    try {
      await exportBackup(credentials)
    } catch (e) {
      setExportError(e instanceof Error ? e.message : 'Export failed')
    } finally {
      setExportBusy(false)
    }
  }

  async function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    setImportError('')
    const file = e.target.files?.[0]
    if (!file) return
    try {
      const text = await file.text()
      const parsed = JSON.parse(text)
      setPendingBackup(parsed)
      setConfirmRestore(true)
    } catch {
      setImportError('Could not parse backup file. Make sure it is a valid AGHSync backup JSON.')
    } finally {
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  async function handleRestore() {
    if (!pendingBackup) return
    setImportError('')
    setImportBusy(true)
    try {
      await importBackup(pendingBackup, credentials)
      setPendingBackup(null)
    } catch (e) {
      setImportError(e instanceof Error ? e.message : 'Restore failed')
    } finally {
      setImportBusy(false)
    }
  }

  if (isLoading) return <p className="text-muted-foreground">Loading…</p>
  if (!settings) return null

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Settings</h1>

      {/* UI Auth */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">UI Authentication</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Status:</span>
            {settings.ui_auth_enabled ? (
              <Badge variant="default">Enabled</Badge>
            ) : (
              <Badge variant="secondary">Disabled</Badge>
            )}
            {settings.ui_auth_enabled && (
              <span className="text-sm text-muted-foreground">(user: {settings.ui_username})</span>
            )}
          </div>

          {settings.ui_auth_enabled ? (
            <Button
              variant="outline"
              onClick={handleDisableAuth}
              disabled={updateUIAuth.isPending}
            >
              {updateUIAuth.isPending ? 'Saving…' : 'Disable Auth'}
            </Button>
          ) : (
            <div className="space-y-3">
              <div className="space-y-1.5">
                <Label htmlFor="auth-username">Username</Label>
                <Input
                  id="auth-username"
                  value={authUsername}
                  onChange={(e) => setAuthUsername(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="auth-password">Password</Label>
                <Input
                  id="auth-password"
                  type="password"
                  value={authPassword}
                  onChange={(e) => setAuthPassword(e.target.value)}
                />
              </div>
              <Button
                onClick={handleEnableAuth}
                disabled={
                  updateUIAuth.isPending || !authUsername || !authPassword
                }
              >
                {updateUIAuth.isPending ? 'Saving…' : 'Enable Auth'}
              </Button>
            </div>
          )}
          {updateUIAuth.isError && (
            <p className="text-sm text-destructive">{updateUIAuth.error.message}</p>
          )}
        </CardContent>
      </Card>

      {/* API Token */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">API Token</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">Status:</span>
            {settings.has_api_token ? (
              <Badge variant="default">Token configured</Badge>
            ) : (
              <Badge variant="secondary">No token</Badge>
            )}
          </div>
          <div className="flex gap-2">
            <Button
              onClick={handleGenerateToken}
              disabled={generateToken.isPending}
            >
              {generateToken.isPending ? 'Generating…' : 'Generate Token'}
            </Button>
            {settings.has_api_token && (
              <Button
                variant="outline"
                onClick={() => setConfirmDeleteToken(true)}
                disabled={deleteToken.isPending}
              >
                Delete Token
              </Button>
            )}
          </div>
          {generateToken.isError && (
            <p className="text-sm text-destructive">{generateToken.error.message}</p>
          )}
        </CardContent>
      </Card>

      {/* New token modal */}
      <Dialog open={tokenModalOpen} onOpenChange={setTokenModalOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Token Generated</DialogTitle>
            <DialogDescription>
              This token will not be shown again. Copy it now.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <pre className="bg-muted rounded p-3 text-xs font-mono break-all whitespace-pre-wrap select-all">
              {newToken}
            </pre>
            <Button size="sm" variant="outline" onClick={copyToken}>
              {copied ? 'Copied!' : 'Copy'}
            </Button>
          </div>
          <DialogFooter>
            <Button onClick={() => { setTokenModalOpen(false); setNewToken('') }}>
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={confirmDeleteToken}
        onOpenChange={setConfirmDeleteToken}
        title="Delete API Token"
        description="Delete the API token? External callers using this token will lose access."
        confirmLabel="Delete"
        onConfirm={() => {
          deleteToken.mutate()
          setConfirmDeleteToken(false)
        }}
        loading={deleteToken.isPending}
      />

      {/* Backup / Restore */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Backup &amp; Restore</CardTitle>
          <p className="text-xs text-muted-foreground mt-1">
            Export all settings — authentication, API token, instances, and sync configuration —
            to a JSON file. Restore replaces the current configuration entirely.
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-1.5">
            <p className="text-sm font-medium">Export</p>
            <Button onClick={handleExport} disabled={exportBusy} variant="outline">
              {exportBusy ? 'Exporting…' : 'Download Backup'}
            </Button>
            {exportError && <p className="text-sm text-destructive">{exportError}</p>}
          </div>

          <div className="space-y-1.5">
            <p className="text-sm font-medium">Restore</p>
            <p className="text-xs text-muted-foreground">
              Importing a backup will replace all current instances and settings.
            </p>
            <Button
              variant="outline"
              onClick={() => fileRef.current?.click()}
              disabled={importBusy}
            >
              {importBusy ? 'Restoring…' : 'Import Backup File…'}
            </Button>
            <input
              ref={fileRef}
              type="file"
              accept=".json,application/json"
              className="hidden"
              onChange={handleFileChange}
            />
            {importError && <p className="text-sm text-destructive">{importError}</p>}
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={confirmRestore}
        onOpenChange={(open) => {
          setConfirmRestore(open)
          if (!open) setPendingBackup(null)
        }}
        title="Restore Backup"
        description="This will replace all current instances and settings with the contents of the backup file. This cannot be undone. Continue?"
        confirmLabel="Restore"
        onConfirm={() => {
          setConfirmRestore(false)
          handleRestore()
        }}
        loading={importBusy}
      />
    </div>
  )
}
