import React, { createContext, useCallback, useContext, useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'aghsync_theme'

interface ThemeContextValue {
  theme: Theme
  toggleTheme: () => void
  applyServerTheme: (serverTheme: string) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

function resolveInitialTheme(): Theme {
  const stored = localStorage.getItem(THEME_STORAGE_KEY)
  if (stored === 'dark' || stored === 'light') return stored
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function applyTheme(theme: Theme) {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(resolveInitialTheme)

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = (e: MediaQueryListEvent) => {
      if (!localStorage.getItem(THEME_STORAGE_KEY)) {
        setThemeState(e.matches ? 'dark' : 'light')
      }
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  const toggleTheme = useCallback(() => {
    setThemeState((prev) => {
      const next: Theme = prev === 'dark' ? 'light' : 'dark'
      localStorage.setItem(THEME_STORAGE_KEY, next)
      return next
    })
  }, [])

  // Called once after settings load to apply the server-saved preference.
  const applyServerTheme = useCallback((serverTheme: string) => {
    if (serverTheme !== 'dark' && serverTheme !== 'light') return
    localStorage.setItem(THEME_STORAGE_KEY, serverTheme)
    setThemeState(serverTheme)
  }, [])

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme, applyServerTheme }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext)
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider')
  return ctx
}
