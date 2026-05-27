import { useEffect, useRef } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import { ThemeProvider, useTheme } from '@/contexts/ThemeContext'
import { AuthProvider, useAuth } from '@/contexts/AuthContext'
import { Layout } from '@/components/Layout'
import { Login } from '@/pages/Login'
import { Dashboard } from '@/pages/Dashboard'
import { Instances } from '@/pages/Instances'
import { Sync } from '@/pages/Sync'
import { History } from '@/pages/History'
import { HistoryRun } from '@/pages/HistoryRun'
import { Settings } from '@/pages/Settings'
import { apiFetch, updateTheme, type Settings as SettingsData } from '@/lib/api'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: 1, staleTime: 30_000 } },
})

// Null-rendering component that syncs theme with the server:
// - On first settings load, applies the server-stored preference (overrides localStorage).
// - When the user toggles, saves the new value to the server (best-effort).
function ThemeServerSync() {
  const { credentials } = useAuth()
  const { theme, applyServerTheme } = useTheme()
  const initialSyncDone = useRef(false)
  const isFirstRender = useRef(true)

  const { data: settings } = useQuery<SettingsData>({
    queryKey: ['settings'],
    queryFn: () => apiFetch<SettingsData>('/api/v1/settings', { credentials }),
  })

  // Apply server theme once on first load.
  useEffect(() => {
    if (settings?.ui_theme && !initialSyncDone.current) {
      initialSyncDone.current = true
      applyServerTheme(settings.ui_theme)
    }
  }, [settings, applyServerTheme])

  // Save theme to server whenever the user toggles it (skip the initial render).
  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false
      return
    }
    updateTheme(theme, credentials).catch(() => { /* best-effort */ })
  }, [theme, credentials])

  return null
}

function AppRoutes() {
  const { authRequired, credentials } = useAuth()

  if (authRequired === null) return null

  if (authRequired && !credentials) {
    return (
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    )
  }

  return (
    <>
      <ThemeServerSync />
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/instances" element={<Instances />} />
          <Route path="/sync" element={<Sync />} />
          <Route path="/history" element={<History />} />
          <Route path="/history/:runId" element={<HistoryRun />} />
          <Route path="/settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </>
  )
}

export function App() {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AuthProvider>
            <AppRoutes />
          </AuthProvider>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  )
}
