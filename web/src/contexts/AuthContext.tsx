import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { type Credentials } from '@/lib/api'

interface AuthState {
  credentials: Credentials | null
  authRequired: boolean | null
}

interface AuthContextValue extends AuthState {
  login: (username: string, password: string) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

const SESSION_KEY = 'aghsync_credentials'

function loadFromSession(): Credentials | null {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    if (!raw) return null
    return JSON.parse(raw) as Credentials
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({ credentials: null, authRequired: null })

  const login = useCallback((username: string, password: string) => {
    const creds: Credentials = { username, password }
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(creds))
    setState({ credentials: creds, authRequired: true })
  }, [])

  const logout = useCallback(() => {
    sessionStorage.removeItem(SESSION_KEY)
    setState({ credentials: null, authRequired: true })
  }, [])

  useEffect(() => {
    const handle = () => {
      sessionStorage.removeItem(SESSION_KEY)
      setState({ credentials: null, authRequired: true })
    }
    window.addEventListener('auth-clear', handle)
    return () => window.removeEventListener('auth-clear', handle)
  }, [])

  useEffect(() => {
    // Probe: GET /api/v1/settings with no credentials to detect auth mode.
    fetch('/api/v1/settings').then((res) => {
      if (res.ok) {
        // Auth disabled — app works with null credentials.
        setState({ credentials: null, authRequired: false })
      } else if (res.status === 401) {
        // Auth required. Restore session if available.
        const saved = loadFromSession()
        setState({ credentials: saved, authRequired: true })
      }
    }).catch(() => {
      // Backend unreachable on initial probe — assume auth required.
      setState({ credentials: null, authRequired: true })
    })
  }, [])

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
